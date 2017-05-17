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
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	kinesisCredentialEnv    = "environment"
	kinesisCredentialStatic = "static"
	kinesisCredentialShared = "shared"
	kinesisCredentialNone   = "none"
)

// Kinesis producer plugin
// This producer sends data to an AWS kinesis stream.
// Configuration example
//
//  - "producer.Kinesis":
//    Region: "eu-west-1"
//    Endpoint: "kinesis.eu-west-1.amazonaws.com"
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
// KinesisStream defines the stream to read from.
// By default this is set to "default"
//
// Region defines the amazon region of your kinesis stream.
// By default this is set to "eu-west-1".
//
// Endpoint defines the amazon endpoint for your kinesis stream.
// By default this is et to "kinesis.eu-west-1.amazonaws.com"
//
// CredentialType defines the credentials that are to be used when
// connecting to kensis. This can be one of the following: environment,
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
// a kinesis record. By default this is set to 500.
//
// RecordMessageDelimiter defines the string to delimit messages within
// a kinesis record. By default this is set to "\n".
//
// SendTimeframeMs defines the timeframe in milliseconds in which a second
// batch send can be triggered. By default this is set to 1000, i.e. one
// send operation per second.
//
// BatchTimeoutSec defines the number of seconds after which a batch is
// flushed automatically. By default this is set to 3.
//
// StreamMapping defines a translation from gollum stream to kinesis stream
// name. If no mapping is given the gollum stream name is used as kinesis
// stream name.
type Kinesis struct {
	core.BufferedProducer `gollumdoc:embed_type`
	client            *kinesis.Kinesis
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
	sequence          *int64
}

const (
	kinesisMetricMessages    = "Kinesis:Messages-"
	kinesisMetricMessagesSec = "Kinesis:MessagesSec-"
)

type streamData struct {
	content            *kinesis.PutRecordsInput
	original           [][]*core.Message
	lastRecordMessages int
}

func init() {
	core.TypeRegistry.Register(Kinesis{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *Kinesis) Configure(conf core.PluginConfigReader) error {
	prod.BufferedProducer.Configure(conf)
	prod.SetStopCallback(prod.close)

	prod.streamMap = conf.GetStreamMap("StreamMapping", "")
	prod.batch = core.NewMessageBatch(conf.GetInt("Batch/MaxMessages", 500))
	prod.recordMaxMessages = conf.GetInt("RecordMaxMessages", 1)
	prod.delimiter = []byte(conf.GetString("RecordMessageDelimiter", "\n"))
	prod.flushFrequency = time.Duration(conf.GetInt("Batch/TimeoutSec", 3)) * time.Second
	prod.sendTimeLimit = time.Duration(conf.GetInt("SendTimeframeMs", 1000)) * time.Millisecond
	prod.lastSendTime = time.Now()
	prod.counters = make(map[string]*int64)
	prod.lastMetricUpdate = time.Now()
	prod.sequence = new(int64)

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
	if endpoint := conf.GetString("Endpoint", "kinesis.eu-west-1.amazonaws.com"); endpoint != "" {
		prod.config.WithEndpoint(endpoint)
	}

	if region := conf.GetString("Region", "eu-west-1"); region != "" {
		prod.config.WithRegion(region)
	}

	// Credentials
	credentialType := strings.ToLower(conf.GetString("Credential/Type", kinesisCredentialNone))
	switch credentialType {
	case kinesisCredentialEnv:
		prod.config.WithCredentials(credentials.NewEnvCredentials())

	case kinesisCredentialStatic:
		id := conf.GetString("Credential/Id", "")
		token := conf.GetString("Credential/Token", "")
		secret := conf.GetString("Credential/Secret", "")
		prod.config.WithCredentials(credentials.NewStaticCredentials(id, secret, token))

	case kinesisCredentialShared:
		filename := conf.GetString("Credential/File", "")
		profile := conf.GetString("Credential/Profile", "")
		prod.config.WithCredentials(credentials.NewSharedCredentials(filename, profile))

	case kinesisCredentialNone:
		// Nothing

	default:
		return fmt.Errorf("Unknown credential type: %s", credentialType)
	}

	for _, streamName := range prod.streamMap {
		metricName := kinesisMetricMessages + streamName
		tgo.Metric.New(metricName)
		tgo.Metric.NewRate(metricName, kinesisMetricMessagesSec+streamName, time.Second, 10, 3, true)
	}

	return conf.Errors.OrNil()
}

func (prod *Kinesis) bufferMessage(msg *core.Message) {
	prod.batch.AppendOrFlush(msg, prod.sendBatch, prod.IsActiveOrStopping, prod.TryFallback)
}

func (prod *Kinesis) sendBatchOnTimeOut() {
	// Flush if necessary
	if prod.batch.ReachedTimeThreshold(prod.flushFrequency) || prod.batch.ReachedSizeThreshold(prod.batch.Len()/2) {
		prod.sendBatch()
	}
}

func (prod *Kinesis) sendBatch() {
	prod.batch.Flush(prod.transformMessages)
}

func (prod *Kinesis) tryFallbackForMessages(messages []*core.Message) {
	for _, msg := range messages {
		prod.TryFallback(msg)
	}
}

func (prod *Kinesis) transformMessages(messages []*core.Message) {
	streamRecords := make(map[core.MessageStreamID]*streamData)

	// Format and sort
	for idx, msg := range messages {
		currentMsg := msg.Clone()
		messageHash := fmt.Sprintf("%X-%d", currentMsg.StreamID(), atomic.AddInt64(prod.sequence, 1))

		// Fetch buffer for this stream
		records, recordsExists := streamRecords[currentMsg.StreamID()]
		if !recordsExists {
			// Select the correct kinesis stream
			streamName, streamMapped := prod.streamMap[currentMsg.StreamID()]
			if !streamMapped {
				streamName, streamMapped = prod.streamMap[core.WildcardStreamID]
				if !streamMapped {
					streamName = core.StreamRegistry.GetStreamName(currentMsg.StreamID())
					prod.streamMap[currentMsg.StreamID()] = streamName

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
			streamRecords[currentMsg.StreamID()] = records
		}

		// Fetch record for this buffer
		var record *kinesis.PutRecordsRequestEntry
		recordExists := len(records.content.Records) > 0
		if !recordExists || records.lastRecordMessages+1 > prod.recordMaxMessages {
			// Append record to stream
			record := &kinesis.PutRecordsRequestEntry{
				Data:         currentMsg.Data(),
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
		record.Data = append(record.Data, msg.Data()...)
		records.lastRecordMessages++
		records.original[len(records.original)-1] = append(records.original[len(records.original)-1], messages[idx])
	}

	sleepDuration := prod.sendTimeLimit - time.Since(prod.lastSendTime)
	if sleepDuration > 0 {
		time.Sleep(sleepDuration)
	}

	// Send to Kinesis
	for _, records := range streamRecords {
		result, err := prod.client.PutRecords(records.content)
		atomic.AddInt64(prod.counters[*records.content.StreamName], int64(len(records.content.Records)))

		if err != nil {
			// Batch failed, fallback all
			prod.Log.Error.Print("Write error: ", err)
			for _, messages := range records.original {
				for _, msg := range messages {
					prod.TryFallback(msg)
				}
			}
		} else {
			// Check each message for errors
			for msgIdx, record := range result.Records {
				if record.ErrorMessage != nil {
					prod.Log.Error.Print("Kinesis message write error: ", *record.ErrorMessage)
					for _, msg := range records.original[msgIdx] {
						prod.TryFallback(msg)
					}
				}
			}
		}
	}
}

func (prod *Kinesis) close() {
	defer prod.WorkerDone()
	prod.DefaultClose()
	prod.batch.Close(prod.transformMessages, prod.GetShutdownTimeout())
}

// Produce writes to stdout or stderr.
func (prod *Kinesis) Produce(workers *sync.WaitGroup) {
	prod.AddMainWorker(workers)

	prod.client = kinesis.New(session.New(prod.config))
	prod.TickerMessageControlLoop(prod.bufferMessage, prod.flushFrequency, prod.sendBatchOnTimeOut)
}
