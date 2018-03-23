// Copyright 2015-2018 trivago N.V.
//
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
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/components"
	"github.com/trivago/tgo"
)

// AwsKinesis producer plugin
//
// This producer sends data to an AWS kinesis stream.
// Configuration example
//
// Parameters
//
// - StreamMapping: This value defines a translation from gollum stream names
// to kinesis stream names. If no mapping is given the gollum stream name is
// used as the kinesis stream name.
// By default this parameter is set to "empty"
//
// - RecordMaxMessages: This value defines the maximum number of messages to join into
// a kinesis record.
// By default this parameter is set to "500".
//
// - RecordMessageDelimiter: This value defines the delimiter string to use between
// messages within a kinesis record.
// By default this parameter is set to "\n".
//
// - SendTimeframeMs: This value defines the timeframe in milliseconds in which a second
// batch send can be triggered.
// By default this parameter is set to "1000".
//
// Examples
//
// This example set up a simple aws Kinesis producer:
//
//  KinesisOut:
//    Type: producer.AwsKinesis
//    Streams: "*"
//    StreamMapping:
//      "*": default
//    Credential:
//      Type: shared
//      File: /Users/<USERNAME>/.aws/credentials
//      Profile: default
//    Region: eu-west-1
//    RecordMaxMessages: 1
//    RecordMessageDelimiter: "\n"
//    SendTimeframeSec: 1
//
type AwsKinesis struct {
	core.BatchedProducer `gollumdoc:"embed_type"`

	// AwsMultiClient is public to make AwsMultiClient.Configure() callable (bug in treflect package)
	AwsMultiClient components.AwsMultiClient `gollumdoc:"embed_type"`

	recordMaxMessages int           `config:"RecordMaxMessages" default:"1"`
	delimiter         []byte        `config:"RecordMessageDelimiter" default:"\n"`
	sendTimeLimit     time.Duration `config:"SendTimeframeMs" default:"1000" metric:"ms"`

	streamMap        map[core.MessageStreamID]string
	client           *kinesis.Kinesis
	lastSendTime     time.Time
	counters         map[string]*int64
	lastMetricUpdate time.Time
	sequence         *int64
}

const (
	kinesisMetricMessages    = "AwsKinesis:Messages-"
	kinesisMetricMessagesSec = "AwsKinesis:MessagesSec-"
)

type streamData struct {
	content            *kinesis.PutRecordsInput
	original           [][]*core.Message
	lastRecordMessages int
}

func init() {
	core.TypeRegistry.Register(AwsKinesis{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *AwsKinesis) Configure(conf core.PluginConfigReader) {
	prod.lastSendTime = time.Now()
	prod.counters = make(map[string]*int64)
	prod.lastMetricUpdate = time.Now()
	prod.sequence = new(int64)

	if prod.recordMaxMessages < 1 {
		prod.recordMaxMessages = 1
		prod.Logger.Warning("RecordMaxMessages was < 1. Defaulting to 1.")
	}

	if prod.recordMaxMessages > 1 && len(prod.delimiter) == 0 {
		prod.delimiter = []byte("\n")
		prod.Logger.Warning("RecordMessageDelimiter was empty. Defaulting to \"\\n\".")
	}

	prod.streamMap = conf.GetStreamMap("StreamMapping", "")
	for _, streamName := range prod.streamMap {
		metricName := kinesisMetricMessages + streamName
		tgo.Metric.New(metricName)
		tgo.Metric.NewRate(metricName, kinesisMetricMessagesSec+streamName, time.Second, 10, 3, true)
	}
}

// Produce writes to stdout or stderr.
func (prod *AwsKinesis) Produce(workers *sync.WaitGroup) {
	defer prod.WorkerDone()

	prod.AddMainWorker(workers)
	prod.initKinesisClient()
	prod.BatchMessageLoop(workers, prod.sendBatch)
}

func (prod *AwsKinesis) initKinesisClient() {
	sess, err := prod.AwsMultiClient.NewSessionWithOptions()
	if err != nil {
		prod.Logger.WithError(err).Error("Can't get proper aws config")
	}

	awsConfig := prod.AwsMultiClient.GetConfig()

	// set auto endpoint to firehose if setting is empty
	if awsConfig.Endpoint == nil || *awsConfig.Endpoint == "" {
		awsConfig.WithEndpoint(fmt.Sprintf("kinesis.%s.amazonaws.com", *awsConfig.Region))
	}

	prod.client = kinesis.New(sess, awsConfig)
}

func (prod *AwsKinesis) sendBatch() core.AssemblyFunc {
	return prod.transformMessages
}

func (prod *AwsKinesis) transformMessages(messages []*core.Message) {
	streamRecords := make(map[core.MessageStreamID]*streamData)

	// Format and sort
	for idx, msg := range messages {
		messageHash := fmt.Sprintf("%X-%d", msg.GetStreamID(), atomic.AddInt64(prod.sequence, 1))

		// Fetch buffer for this stream
		records, recordsExists := streamRecords[msg.GetStreamID()]
		if !recordsExists {
			// Select the correct kinesis stream
			streamName, streamMapped := prod.streamMap[msg.GetStreamID()]
			if !streamMapped {
				streamName, streamMapped = prod.streamMap[core.WildcardStreamID]
				if !streamMapped {
					streamName = core.StreamRegistry.GetStreamName(msg.GetStreamID())
					prod.streamMap[msg.GetStreamID()] = streamName

					metricName := kinesisMetricMessages + streamName
					tgo.Metric.New(metricName)
					tgo.Metric.NewRate(metricName, kinesisMetricMessagesSec+streamName, time.Second, 10, 3, true)
				}
			}

			// Create buffers for this kinesis stream
			maxLength := len(messages)/prod.recordMaxMessages + 1
			records = &streamData{
				content: &kinesis.PutRecordsInput{
					Records:    make([]*kinesis.PutRecordsRequestEntry, 0, maxLength),
					StreamName: aws.String(streamName),
				},
				original:           make([][]*core.Message, 0, maxLength),
				lastRecordMessages: 0,
			}
			streamRecords[msg.GetStreamID()] = records
		}

		// Fetch record for this buffer
		var record *kinesis.PutRecordsRequestEntry
		recordExists := len(records.content.Records) > 0
		if !recordExists || records.lastRecordMessages+1 > prod.recordMaxMessages {
			// Append record to stream
			record = &kinesis.PutRecordsRequestEntry{
				Data:         msg.GetPayload(),
				PartitionKey: aws.String(messageHash),
			}
			records.content.Records = append(records.content.Records, record)
			records.original = append(records.original, make([]*core.Message, 0, prod.recordMaxMessages))
			records.lastRecordMessages = 0
		} else {
			record = records.content.Records[len(records.content.Records)-1]
			record.Data = append(record.Data, prod.delimiter...)
		}

		// Append message to record
		record.Data = append(record.Data, msg.GetPayload()...)
		records.lastRecordMessages++
		records.original[len(records.original)-1] = append(records.original[len(records.original)-1], messages[idx])
	}

	sleepDuration := prod.sendTimeLimit - time.Since(prod.lastSendTime)
	if sleepDuration > 0 {
		time.Sleep(sleepDuration)
	}

	// Send to AwsKinesis
	for _, records := range streamRecords {
		result, err := prod.client.PutRecords(records.content)
		atomic.AddInt64(prod.counters[*records.content.StreamName], int64(len(records.content.Records)))

		if err != nil {
			// Batch failed, fallback all
			prod.Logger.WithError(err).Error("Failed to put records")
			for _, messages := range records.original {
				for _, msg := range messages {
					prod.TryFallback(msg)
				}
			}
		} else {
			// Check each message for errors
			for msgIdx, record := range result.Records {
				if record.ErrorMessage != nil {
					prod.Logger.Error("AwsKinesis message write error: ", *record.ErrorMessage)
					for _, msg := range records.original[msgIdx] {
						prod.TryFallback(msg)
					}
				}
			}
		}
	}
}
