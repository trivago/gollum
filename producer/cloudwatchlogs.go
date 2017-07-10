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
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/trivago/gollum/core"
	"time"
)

// AWS CloudWatch specific constants.
// Also see http://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/cloudwatch_limits_cwl.html
const (
	// Maximum number of log events in a batch.
	maxBatchEvents = 10000
	// Maximum batch size in bytes.
	maxBatchSize = 1048576
	// Maximum event size in bytes.
	maxEventSize = 262144
	// A batch of log events in a single PutLogEvents request cannot span more than 24 hours.
	maxBatchTimeSpan = 24 * time.Hour
	// How many bytes to append to each log event.
	eventSizeOverhead = 26
	// DescribeLogStreams transactions/second.
	describeLogstreamsDelay = 200 * time.Millisecond
	// PutLogEvents 5 requests/second/log stream.
	putLogEventsDelay = 200 * time.Millisecond
)

type CloudwatchLogs struct {
	core.BufferedProducer `gollumdoc:"embed_type"`
	stream                *string
	group                 *string
	token                 *string
	service               *cloudwatchlogs.CloudWatchLogs
}

func init() {
	core.TypeRegistry.Register(CloudwatchLogs{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *CloudwatchLogs) Configure(conf core.PluginConfigReader) error {
	prod.SetStopCallback(prod.flush)
	stream := conf.GetString("stream", "")
	if stream == "" {
		return fmt.Errorf("stream name can not be empty")
	}
	group := conf.GetString("group", "")
	if group == "" {
		return fmt.Errorf("group name can not be empty")
	}
	prod.stream = &stream
	prog.group = &group
	return nil
}

// Put log events and update sequence token.
// Possible errors http://docs.aws.amazon.com/AmazonCloudWatchLogs/latest/APIReference/API_PutLogEvents.html
func (prod *CloudwatchLogs) upload() {
	params := &cloudwatchlogs.PutLogEventsInput{
		LogEvents:     logevents,
		LogGroupName:  prod.group,
		LogStreamName: prod.stream,
		SequenceToken: prod.token,
	}
	// When rejectedLogEventsInfo is not empty, app can not
	// do anything reasonable with rejected logs. Ignore it.
	// Meybe expose some statistics for rejected counters.
	resp, err := dst.svc.PutLogEvents(prod.cloudwatchLogsParams)
	if err == nil {
		prod.cloudwatchLogsParams.SequenceToken = resp.NextSequenceToken
	}
}

func (prod *Console) Produce(workers *sync.WaitGroup) {
	defer prod.WorkerDone()
	prod.AddMainWorker(workers)
	prod.MessageControlLoop(prod.printMessage)
}

// For newly created log streams, token is an empty string.
func (prod *CloudwatchLogs) setToken() error {
	params := &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName:        prod.group,
		LogStreamNamePrefix: prod.stream,
	}

	return prod.service.DescribeLogStreamsPages(params,
		func(page *cloudwatchlogs.DescribeLogStreamsOutput, lastPage bool) bool {
			return !findToken(prod, page)
		})
}

func findToken(prod *CloudwatchLogs, page *cloudwatchlogs.DescribeLogStreamsOutput) bool {
	for _, row := range page.LogStreams {
		if *prod.stream == *row.LogStreamName {
			prod.token = row.UploadSequenceToken
			return true
		}
	}
	return false
}

// Create log group and stream. If an error is returned, PutLogEvents cannot succeed.
func (prod *CloudwatchLogs) create() (err error) {
	err = prod.createGroup()
	if err != nil {
		return
	}
	err = prod.createStream()
	return
}

// http://docs.aws.amazon.com/AmazonCloudWatchLogs/latest/APIReference/API_CreateLogGroup.html
func (prod *CloudwatchLogs) createGroup() error {
	params := &cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: &prod.group,
	}
	_, err := prod.service.CreateLogGroup(params)
	if err, ok := err.(awserr.Error); ok {
		if err.Code() == "ResourceAlreadyExistsException" {
			return nil
		}
	}
	return err
}

// http://docs.aws.amazon.com/AmazonCloudWatchLogs/latest/APIReference/API_CreateLogStream.html
func (prod *CloudwatchLogs) createStream() error {
	params := &cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  &prod.group,
		LogStreamName: &prod.stream,
	}
	_, err := prod.service.CreateLogStream(params)
	if err, ok := err.(awserr.Error); ok {
		if err.Code() == "ResourceAlreadyExistsException" {
			return nil
		}
	}
	return err
}
