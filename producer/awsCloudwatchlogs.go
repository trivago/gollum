// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package producer

import (
	"sort"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/components"
)

// AWS CloudWatch specific constants.
// Also see http://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/cloudwatch_limits_cwl.html
const (
	// Maximum number of log events in a batch.
	maxBatchEvents = 10000
	// Maximum batch size in bytes.
	maxBatchSize = 1048576
	// Maximum event size in bytes.
	//maxEventSize = 262144
	// A batch of log events in a single PutLogEvents request cannot span more than 24 hours.
	maxBatchTimeSpan = 24 * time.Hour
	// How many bytes to append to each log event.
	eventSizeOverhead = 26
	// DescribeLogStreams transactions/second.
	//describeLogstreamsDelay = 200 * time.Millisecond
	// PutLogEvents 5 requests/second/log stream.
	//putLogEventsDelay = 200 * time.Millisecond
)

// AwsCloudwatchLogs producer
//
// The AwsCloudwatchLogs producer plugin sends messages to
// AWS Cloudwatch Logs service. Credentials are obtained by gollum automaticly.
//
// Patameters
//
// - LogStream: Stream name in cloudwatch logs.
//
// - LogGroup: Group name in cloudwatch logs.
//
// - Region: Amazon region into which stream logs to. Defaults to "eu-west-1".
//
// Examples
//
// This configuration sends messages to stream stream_name and group group_name with shared credentials.
//
// CwLogs:
//  Type: AwsCloudwatchLogs:
//      LogStream: stream_name
//      LogGroup: group_name
//  Credential:
//      Type: shared
type AwsCloudwatchLogs struct {
	AwsMultiClient       components.AwsMultiClient `gollumdoc:"embed_type"`
	core.BatchedProducer `gollumdoc:"embed_type"`
	stream               string `config:"LogStream" default:""`
	group                string `config:"LogGroup" default:""`
	token                *string
	service              *cloudwatchlogs.CloudWatchLogs
}

func init() {
	core.TypeRegistry.Register(AwsCloudwatchLogs{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *AwsCloudwatchLogs) Configure(conf core.PluginConfigReader) {
	if prod.stream == "" {
		prod.Logger.Error("LogStream can not be empty")
	}
	if prod.group == "" {
		prod.Logger.Error("LogGroup can not be empty")
	}
	if conf.GetInt("Batch/MaxCount", maxBatchEvents) > maxBatchEvents {
		conf.Errors.Pushf("Batch/MaxCount must be below %d", maxBatchEvents)
	}
}

func (m byTimestamp) Len() int {
	return len(m)
}

func (m byTimestamp) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

func (m byTimestamp) Less(i, j int) bool {
	return m[i].GetCreationTime().Unix() < m[j].GetCreationTime().Unix()
}

type byTimestamp []*core.Message

// Return lowest index based on all check functions
// This function assumes that messages are sorted by timestamp in ascending order
func (prod *AwsCloudwatchLogs) numEvents(messages []*core.Message, checkFn ...indexNumFn) int {
	index := maxBatchEvents
	for _, fn := range checkFn {
		result := fn(messages)
		if result < index {
			index = result
		}
	}
	return index
}

type indexNumFn func(messages []*core.Message) int

// Return lowest index based on message size
func (prod *AwsCloudwatchLogs) sizeIndex(messages []*core.Message) int {
	size, index := 0, 0
	for i, message := range messages {
		size += len(message.String()) + eventSizeOverhead
		if size > maxBatchSize {
			break
		}
		index = i + 1
	}
	return index
}

// Return lowest index based on timespan
// This function assumes that messages are sorted by timestamp in ascending order
func (prod *AwsCloudwatchLogs) timeIndex(messages []*core.Message) (index int) {
	if len(messages) == 0 {
		return 0
	}
	firstTimestamp := messages[0].GetCreationTime().Unix()
	for i, message := range messages {
		if (message.GetCreationTime().Unix() - firstTimestamp) > int64(maxBatchTimeSpan) {
			break
		}
		index = i + 1
	}
	return index
}

func (prod *AwsCloudwatchLogs) processBatch(messages []*core.Message) []*core.Message {
	sort.Sort(byTimestamp(messages))
	return messages[:prod.numEvents(messages, prod.sizeIndex, prod.timeIndex)]
}

// Put log events and update sequence token.
// Possible errors http://docs.aws.amazon.com/AmazonCloudWatchLogs/latest/APIReference/API_PutLogEvents.html
func (prod *AwsCloudwatchLogs) upload(messages []*core.Message) {
	messages = prod.processBatch(messages)
	if len(messages) == 0 {
		return
	}
	logevents := make([]*cloudwatchlogs.InputLogEvent, 0, len(messages))
	for _, msg := range messages {
		logevents = append(logevents, &cloudwatchlogs.InputLogEvent{
			Message:   aws.String(msg.String()),
			Timestamp: aws.Int64(msg.GetCreationTime().Unix() * 1000),
		})
	}
	params := &cloudwatchlogs.PutLogEventsInput{
		LogEvents:     logevents,
		LogGroupName:  aws.String(prod.group),
		LogStreamName: aws.String(prod.stream),
		SequenceToken: prod.token,
	}
	prod.handleResult(prod.service.PutLogEvents(params))
}

// When rejectedLogEventsInfo is not empty, app can not
// do anything reasonable with rejected logs. Ignore it.
func (prod *AwsCloudwatchLogs) handleResult(resp *cloudwatchlogs.PutLogEventsOutput, result error) {
	switch err := result.(type) {
	case awserr.Error:
		switch err.Code() {
		case "InvalidSequenceTokenException":
			prod.Logger.Debugf("invalid sequence token, updating")
			prod.setToken()
		case "ResourceNotFoundException":
			prod.Logger.Debugf("missing group/stream, creating")
			prod.create()
			prod.token = nil
		default:
			prod.Logger.Errorf("error while sending batch. code: %s message: %q", err.Code(), err.Message())
		}
	case nil:
		prod.token = resp.NextSequenceToken
	default:
		prod.Logger.Errorf("error while sending batch: %q", err)
	}
}

func (prod *AwsCloudwatchLogs) sendBatch() core.AssemblyFunc {
	return prod.upload
}

// Produce starts the producer
func (prod *AwsCloudwatchLogs) Produce(workers *sync.WaitGroup) {
	prod.initService()
	if err := prod.create(); err != nil {
		prod.Logger.Errorf("could not create group:%q stream:%q error was: %q", prod.group, prod.stream, err)
	} else {
		prod.setToken()
	}
	prod.BatchMessageLoop(workers, prod.sendBatch)
}

func (prod *AwsCloudwatchLogs) initService() {
	sess, err := prod.AwsMultiClient.NewSessionWithOptions()
	if err != nil {
		prod.Logger.WithError(err).Error("Can't get proper aws config")
	}

	awsConfig := prod.AwsMultiClient.GetConfig()
	// Fill in nedpoint URL if not provided
	if awsConfig.Endpoint == nil || *awsConfig.Endpoint == "" {
		resolver := endpoints.DefaultResolver()
		endpoint, err := resolver.EndpointFor(endpoints.LogsServiceID, *awsConfig.Region)
		if err != nil {
			prod.Logger.WithError(err).Error("Can't resolve cloudwatch logs endpoint URL")
		}
		awsConfig.WithEndpoint(endpoint.URL)
	}
	prod.service = cloudwatchlogs.New(sess, awsConfig)
}

// For newly created log streams, token is an empty string.
func (prod *AwsCloudwatchLogs) setToken() error {
	params := &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName:        &prod.group,
		LogStreamNamePrefix: &prod.stream,
	}

	return prod.service.DescribeLogStreamsPages(params,
		func(page *cloudwatchlogs.DescribeLogStreamsOutput, lastPage bool) bool {
			return !findToken(prod, page)
		})
}

func findToken(prod *AwsCloudwatchLogs, page *cloudwatchlogs.DescribeLogStreamsOutput) bool {
	for _, row := range page.LogStreams {
		if prod.stream == *row.LogStreamName {
			prod.token = row.UploadSequenceToken
			return true
		}
	}
	return false
}

// Create log group and stream. If an error is returned, PutLogEvents cannot succeed.
func (prod *AwsCloudwatchLogs) create() error {
	if err := prod.createGroup(); err != nil {
		return err
	}
	return prod.createStream()
}

// http://docs.aws.amazon.com/AmazonCloudWatchLogs/latest/APIReference/API_CreateLogGroup.html
func (prod *AwsCloudwatchLogs) createGroup() error {
	params := &cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: &prod.group,
	}
	_, err := prod.service.CreateLogGroup(params)
	if awsErr, ok := err.(awserr.Error); ok {
		if awsErr.Code() == "ResourceAlreadyExistsException" {
			return nil
		}
	}
	return err
}

// http://docs.aws.amazon.com/AmazonCloudWatchLogs/latest/APIReference/API_CreateLogStream.html
func (prod *AwsCloudwatchLogs) createStream() error {
	params := &cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  &prod.group,
		LogStreamName: &prod.stream,
	}
	_, err := prod.service.CreateLogStream(params)
	if awsErr, ok := err.(awserr.Error); ok {
		if awsErr.Code() == "ResourceAlreadyExistsException" {
			return nil
		}
	}
	return err
}
