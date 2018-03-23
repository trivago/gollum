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
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/components"
	"github.com/trivago/tgo"
)

// AwsFirehose producer plugin
//
// This producer sends data to an AWS Firehose stream.
//
// Parameters
//
// - StreamMapping: This value defines a translation from gollum stream names
// to firehose stream names. If no mapping is given, the gollum stream name is
// used as the firehose stream name.
// By default this parameter is set to "empty"
//
// - RecordMaxMessages: This value defines the number of messages to send
// in one record to aws firehose.
// By default this parameter is set to "1".
//
// - RecordMessageDelimiter: This value defines the delimiter string to use between
// messages within a firehose record.
// By default this parameter is set to "\n".
//
// - SendTimeframeMs: This value defines the timeframe in milliseconds in which a second
// batch send can be triggered.
// By default this parameter is set to "1000".
//
// Examples
//
// This example set up a simple aws firehose producer:
//
//  firehoseOut:
//    Type: producer.AwsFirehose
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
type AwsFirehose struct {
	core.BatchedProducer `gollumdoc:"embed_type"`

	// AwsMultiClient is public to make AwsMultiClient.Configure() callable (bug in treflect package)
	AwsMultiClient components.AwsMultiClient `gollumdoc:"embed_type"`

	recordMaxMessages int           `config:"RecordMaxMessages" default:"1"`
	delimiter         []byte        `config:"RecordMessageDelimiter" default:"\n"`
	sendTimeLimit     time.Duration `config:"SendTimeframeMs" default:"1000" metric:"ms"`

	client           *firehose.Firehose
	streamMap        map[core.MessageStreamID]string
	lastSendTime     time.Time
	lastMetricUpdate time.Time
	counters         map[string]*int64
}

const (
	firehoseMetricMessages    = "AwsFirehose:Messages-"
	firehoseMetricMessagesSec = "AwsFirehose:MessagesSec-"
)

type firehoseData struct {
	content            *firehose.PutRecordBatchInput
	original           [][]*core.Message
	lastRecordMessages int
}

func init() {
	core.TypeRegistry.Register(AwsFirehose{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *AwsFirehose) Configure(conf core.PluginConfigReader) {
	prod.streamMap = conf.GetStreamMap("StreamMapping", "default")

	prod.lastSendTime = time.Now()
	prod.counters = make(map[string]*int64)
	prod.lastMetricUpdate = time.Now()

	if prod.recordMaxMessages < 1 {
		prod.recordMaxMessages = 1
		prod.Logger.Warning("RecordMaxMessages was < 1. Defaulting to 1.")
	}

	if prod.recordMaxMessages > 1 && len(prod.delimiter) == 0 {
		prod.delimiter = []byte("\n")
		prod.Logger.Warning("RecordMessageDelimiter was empty. Defaulting to \"\\n\".")
	}

	for _, streamName := range prod.streamMap {
		tgo.Metric.New(firehoseMetricMessages + streamName)
		tgo.Metric.New(firehoseMetricMessagesSec + streamName)
		prod.counters[streamName] = new(int64)
	}
}

// Produce writes to stdout or stderr.
func (prod *AwsFirehose) Produce(workers *sync.WaitGroup) {
	defer prod.WorkerDone()

	prod.AddMainWorker(workers)
	prod.initFirehoseClient()
	prod.BatchMessageLoop(workers, prod.sendBatch)
}

func (prod *AwsFirehose) initFirehoseClient() {
	sess, err := prod.AwsMultiClient.NewSessionWithOptions()
	if err != nil {
		prod.Logger.WithError(err).Error("Can't get proper aws config")
	}

	awsConfig := prod.AwsMultiClient.GetConfig()

	// set auto endpoint to firehose if setting is empty
	if awsConfig.Endpoint == nil || *awsConfig.Endpoint == "" {
		awsConfig.WithEndpoint(fmt.Sprintf("firehose.%s.amazonaws.com", *awsConfig.Region))
	}

	prod.client = firehose.New(sess, awsConfig)
}

func (prod *AwsFirehose) sendBatch() core.AssemblyFunc {
	return prod.transformMessages
}

func (prod *AwsFirehose) transformMessages(messages []*core.Message) {
	streamRecords := make(map[core.MessageStreamID]*firehoseData)

	// Format and sort
	for idx, msg := range messages {

		// Fetch buffer for this stream
		records, recordsExists := streamRecords[msg.GetStreamID()]
		if !recordsExists {
			// Select the correct firehose stream
			streamName, streamMapped := prod.streamMap[msg.GetStreamID()]
			if !streamMapped {
				streamName = core.StreamRegistry.GetStreamName(msg.GetStreamID())
				prod.streamMap[msg.GetStreamID()] = streamName

				tgo.Metric.New(firehoseMetricMessages + streamName)
				tgo.Metric.New(firehoseMetricMessagesSec + streamName)
				prod.counters[streamName] = new(int64)
			}

			// Create buffers for this firehose stream
			maxLength := len(messages)/prod.recordMaxMessages + 1
			records = &firehoseData{
				content: &firehose.PutRecordBatchInput{
					Records:            make([]*firehose.Record, 0, maxLength),
					DeliveryStreamName: aws.String(streamName),
				},
				original:           make([][]*core.Message, 0, maxLength),
				lastRecordMessages: 0,
			}
			streamRecords[msg.GetStreamID()] = records
		}

		// Fetch record for this buffer
		var record *firehose.Record
		recordExists := len(records.content.Records) > 0
		if !recordExists || records.lastRecordMessages+1 > prod.recordMaxMessages {
			// Append record to stream
			record = &firehose.Record{
				Data: make([]byte, 0, len(msg.GetPayload())),
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

	// Send to AwsFirehose
	for _, records := range streamRecords {
		rsp, err := prod.client.PutRecordBatch(records.content)
		atomic.AddInt64(prod.counters[*records.content.DeliveryStreamName], int64(len(records.content.Records)))

		if err != nil {
			// Batch failed, fallback all
			prod.Logger.WithError(err).Error("Failed to put record batch")
			for _, messages := range records.original {
				for _, msg := range messages {
					prod.TryFallback(msg)
				}
			}
		} else {
			// Check each message for errors
			for msgIdx, record := range rsp.RequestResponses {
				if record.ErrorMessage != nil {
					prod.Logger.Error("AwsFirehose message write error: ", *record.ErrorMessage)
					for _, msg := range records.original[msgIdx] {
						prod.TryFallback(msg)
					}
				}
			}
		}
	}
}
