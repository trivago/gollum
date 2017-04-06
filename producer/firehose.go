// Copyright 2015-2017 trivago GmbH
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
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	firehoseCredentialEnv    = "environment"
	firehoseCredentialStatic = "static"
	firehoseCredentialShared = "shared"
	firehoseCredentialNone   = "none"
)

// Firehose producer plugin
// This producer sends data to an AWS Firehose stream.
// Configuration example
//
//  - "producer.Firehose":
//    Region: "eu-west-1"
//    Endpoint: "firehose.eu-west-1.amazonaws.com"
//    CredentialType: "none"
//    CredentialId: ""
//    CredentialToken: ""
//    CredentialSecret: ""
//    CredentialFile: ""
//    CredentialProfile: ""
//    BatchMaxMessages: 500
//    RecordMaxMessages: 1
//    RecordMessageDelimiter: "\n"
//    SendTimeframeSec: 1
//    BatchTimeoutSec: 3
//    StreamMapping:
//      "*" : "default"
//
// Firehose defines the stream to read from.
// By default this is set to "default"
//
// Region defines the amazon region of your firehose stream.
// By default this is set to "eu-west-1".
//
// Endpoint defines the amazon endpoint for your firehose stream.
// By default this is set to "firehose.eu-west-1.amazonaws.com"
//
// CredentialType defines the credentials that are to be used when
// connecting to firehose. This can be one of the following: environment,
// static, shared, none.
// Static enables the parameters CredentialId, CredentialToken and
// CredentialSecret shared enables the parameters CredentialFile and
// CredentialProfile. None will not use any credentials and environment
// will pull the credentials from environmental settings.
// By default this is set to none.
//
// BatchMaxMessages defines the maximum number of messages to send per
// batch. By default this is set to 500.
//
// RecordMaxMessages defines the maximum number of messages to join into
// a firehose record. By default this is set to 500.
//
// RecordMessageDelimiter defines the string to delimit messages within
// a firehose record. By default this is set to "\n".
//
// SendTimeframeMs defines the timeframe in milliseconds in which a second
// batch send can be triggered. By default this is set to 1000, i.e. one
// send operation per second.
//
// BatchTimeoutSec defines the number of seconds after which a batch is
// flushed automatically. By default this is set to 3.
//
// StreamMapping defines a translation from gollum stream to firehose stream
// name. If no mapping is given the gollum stream name is used as firehose
// stream name.
type Firehose struct {
	core.BufferedProducer
	client            *firehose.Firehose
	config            *aws.Config
	streamMap         map[core.MessageStreamID]string
	batch             core.MessageBatch
	recordMaxMessages int
	delimiter         []byte
	flushFrequency    time.Duration
	lastSendTime      time.Time
	sendTimeLimit     time.Duration
	counters          map[string]*int64
	lastMetricUpdate  time.Time
}

const (
	firehoseMetricMessages    = "Firehose:Messages-"
	firehoseMetricMessagesSec = "Firehose:MessagesSec-"
)

type firehoseData struct {
	content            *firehose.PutRecordBatchInput
	original           [][]*core.Message
	lastRecordMessages int
}

func init() {
	core.TypeRegistry.Register(Firehose{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *Firehose) Configure(conf core.PluginConfigReader) error {
	prod.SimpleProducer.Configure(conf)
	prod.SetStopCallback(prod.close)

	prod.streamMap = conf.GetStreamMap("StreamMapping", "default")
	prod.batch = core.NewMessageBatch(conf.GetInt("BatchMaxMessages", 500))
	prod.recordMaxMessages = conf.GetInt("RecordMaxMessages", 1)
	prod.delimiter = []byte(conf.GetString("RecordMessageDelimiter", "\n"))
	prod.flushFrequency = time.Duration(conf.GetInt("BatchTimeoutSec", 3)) * time.Second
	prod.sendTimeLimit = time.Duration(conf.GetInt("SendTimeframeMs", 1000)) * time.Millisecond
	prod.lastSendTime = time.Now()
	prod.counters = make(map[string]*int64)
	prod.lastMetricUpdate = time.Now()

	if prod.recordMaxMessages < 1 {
		prod.recordMaxMessages = 1
		prod.Log.Warning.Print("RecordMaxMessages was < 1. Defaulting to 1.")
	}

	if prod.recordMaxMessages > 1 && len(prod.delimiter) == 0 {
		prod.delimiter = []byte("\n")
		prod.Log.Warning.Print("RecordMessageDelimiter was empty. Defaulting to \"\\n\".")
	}

	// Config
	prod.config = aws.NewConfig()
	if endpoint := conf.GetString("Endpoint", "firehose.eu-west-1.amazonaws.com"); endpoint != "" {
		prod.config.WithEndpoint(endpoint)
	}

	if region := conf.GetString("Region", "eu-west-1"); region != "" {
		prod.config.WithRegion(region)
	}

	// Credentials
	credentialType := strings.ToLower(conf.GetString("CredentialType", firehoseCredentialNone))
	switch credentialType {
	case firehoseCredentialEnv:
		prod.config.WithCredentials(credentials.NewEnvCredentials())

	case firehoseCredentialStatic:
		id := conf.GetString("CredentialId", "")
		token := conf.GetString("CredentialToken", "")
		secret := conf.GetString("CredentialSecret", "")
		prod.config.WithCredentials(credentials.NewStaticCredentials(id, secret, token))

	case firehoseCredentialShared:
		filename := conf.GetString("CredentialFile", "")
		profile := conf.GetString("CredentialProfile", "")
		prod.config.WithCredentials(credentials.NewSharedCredentials(filename, profile))

	case firehoseCredentialNone:
		// Nothing

	default:
		return fmt.Errorf("Unknown CredentialType: %s", credentialType)
	}

	for _, streamName := range prod.streamMap {
		tgo.Metric.New(firehoseMetricMessages + streamName)
		tgo.Metric.New(firehoseMetricMessagesSec + streamName)
		prod.counters[streamName] = new(int64)
	}

	return nil
}

func (prod *Firehose) bufferMessage(msg *core.Message) {
	prod.batch.AppendOrFlush(msg, prod.sendBatch, prod.IsActiveOrStopping, prod.Drop)
}

func (prod *Firehose) sendBatchOnTimeOut() {
	// Flush if necessary
	if prod.batch.ReachedTimeThreshold(prod.flushFrequency) || prod.batch.ReachedSizeThreshold(prod.batch.Len()/2) {
		prod.sendBatch()
	}

	duration := time.Since(prod.lastMetricUpdate)
	prod.lastMetricUpdate = time.Now()

	for streamName, counter := range prod.counters {
		count := atomic.SwapInt64(counter, 0)

		tgo.Metric.Add(firehoseMetricMessages+streamName, count)
		tgo.Metric.SetF(firehoseMetricMessagesSec+streamName, float64(count)/duration.Seconds())
	}
}

func (prod *Firehose) sendBatch() {
	prod.batch.Flush(prod.transformMessages)
}

func (prod *Firehose) dropMessages(messages []*core.Message) {
	for _, msg := range messages {
		prod.Drop(msg)
	}
}

func (prod *Firehose) transformMessages(messages []*core.Message) {
	streamRecords := make(map[core.MessageStreamID]*firehoseData)

	// Format and sort
	for idx, msg := range messages {

		// Fetch buffer for this stream
		records, recordsExists := streamRecords[msg.StreamID()]
		if !recordsExists {
			// Select the correct firehose stream
			streamName, streamMapped := prod.streamMap[msg.StreamID()]
			if !streamMapped {
				streamName = core.StreamRegistry.GetStreamName(msg.StreamID())
				prod.streamMap[msg.StreamID()] = streamName

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
			streamRecords[msg.StreamID()] = records
		}

		// Fetch record for this buffer
		var record *firehose.Record
		recordExists := len(records.content.Records) > 0
		if !recordExists || records.lastRecordMessages+1 > prod.recordMaxMessages {
			// Append record to stream
			record = &firehose.Record{
				Data: make([]byte, 0, len(msg.Data())),
			}
			records.content.Records = append(records.content.Records, record)
			records.original = append(records.original, make([]*core.Message, 0, prod.recordMaxMessages))
			records.lastRecordMessages = 0
		} else {
			record = records.content.Records[len(records.content.Records)-1]
			record.Data = append(record.Data, prod.delimiter...)
		}

		// Append message to record
		record.Data = append(record.Data, msg.Data()...)
		records.lastRecordMessages++
		records.original[len(records.original)-1] = append(records.original[len(records.original)-1], messages[idx])
	}

	sleepDuration := prod.sendTimeLimit - time.Since(prod.lastSendTime)
	if sleepDuration > 0 {
		time.Sleep(sleepDuration)
	}

	// Send to Firehose
	for _, records := range streamRecords {
		result, err := prod.client.PutRecordBatch(records.content)
		atomic.AddInt64(prod.counters[*records.content.DeliveryStreamName], int64(len(records.content.Records)))

		if err != nil {
			// Batch failed, drop all
			prod.Log.Error.Print("Firehose write error: ", err)
			for _, messages := range records.original {
				for _, msg := range messages {
					prod.Drop(msg)
				}
			}
		} else {
			// Check each message for errors
			for msgIdx, record := range result.RequestResponses {
				if record.ErrorMessage != nil {
					prod.Log.Error.Print("Firehose message write error: ", *record.ErrorMessage)
					for _, msg := range records.original[msgIdx] {
						prod.Drop(msg)
					}
				}
			}
		}
	}
}

func (prod *Firehose) close() {
	defer prod.WorkerDone()
	prod.CloseMessageChannel(prod.bufferMessage)
	prod.batch.Close(prod.transformMessages, prod.GetShutdownTimeout())
}

// Produce writes to stdout or stderr.
func (prod *Firehose) Produce(workers *sync.WaitGroup) {
	prod.AddMainWorker(workers)

	prod.client = firehose.New(session.New(prod.config))
	prod.TickerMessageControlLoop(prod.bufferMessage, prod.flushFrequency, prod.sendBatchOnTimeOut)
}
