// Copyright 2015-2016 trivago GmbH
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
//    SendTimeframeSec: 1
//    BatchTimeoutSec: 3
//    TimoutMs: 1500
//    StreamMap:
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
// connectiong to kensis. This can be one of the following: environment,
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
// SendTimeframeMs defines the timeframe in milliseconds in which a second
// batch send can be triggered. By default this is set to 1000, i.e. one
// send operation per second.
//
// BatchTimeoutSec defines the number of seconds after which a batch is
// flushed automatically. By default this is set to 3.
type Kinesis struct {
	core.ProducerBase
	client           *kinesis.Kinesis
	config           *aws.Config
	streamMap        map[core.MessageStreamID]string
	batch            core.MessageBatch
	flushFrequency   time.Duration
	lastSendTime     time.Time
	sendTimeLimit    time.Duration
	counters         map[string]*int64
	lastMetricUpdate time.Time
}

const (
	kinesisMetricMessages    = "Kinesis:Messages-"
	kinesisMetricMessagesSec = "Kinesis:MessagesSec-"
)

type streamData struct {
	content  *kinesis.PutRecordsInput
	original []*core.Message
}

func init() {
	core.TypeRegistry.Register(Kinesis{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *Kinesis) Configure(conf core.PluginConfigReader) error {
	prod.ProducerBase.Configure(conf)
	prod.SetStopCallback(prod.close)

	prod.streamMap = conf.GetStreamMap("StreamMapping", "default")
	prod.batch = core.NewMessageBatch(conf.GetInt("BatchMaxMessages", 500))
	prod.flushFrequency = time.Duration(conf.GetInt("BatchTimeoutSec", 3)) * time.Second
	prod.sendTimeLimit = time.Duration(conf.GetInt("SendTimeframeMs", 1000)) * time.Millisecond
	prod.lastSendTime = time.Now()
	prod.counters = make(map[string]*int64)
	prod.lastMetricUpdate = time.Now()

	// Config
	prod.config = aws.NewConfig()
	if endpoint := conf.GetString("Endpoint", "kinesis.eu-west-1.amazonaws.com"); endpoint != "" {
		prod.config.WithEndpoint(endpoint)
	}

	if region := conf.GetString("Region", "eu-west-1"); region != "" {
		prod.config.WithRegion(region)
	}

	// Credentials
	credentialType := strings.ToLower(conf.GetString("CredentialType", kinesisCredentialNone))
	switch credentialType {
	case kinesisCredentialEnv:
		prod.config.WithCredentials(credentials.NewEnvCredentials())

	case kinesisCredentialStatic:
		id := conf.GetString("CredentialId", "")
		token := conf.GetString("CredentialToken", "")
		secret := conf.GetString("CredentialSecret", "")
		prod.config.WithCredentials(credentials.NewStaticCredentials(id, secret, token))

	case kinesisCredentialShared:
		filename := conf.GetString("CredentialFile", "")
		profile := conf.GetString("CredentialProfile", "")
		prod.config.WithCredentials(credentials.NewSharedCredentials(filename, profile))

	case kinesisCredentialNone:
		// Nothing

	default:
		return fmt.Errorf("Unknown CredentialType: %s", credentialType)
	}

	for _, streamName := range prod.streamMap {
		tgo.Metric.New(kinesisMetricMessages + streamName)
		tgo.Metric.New(kinesisMetricMessagesSec + streamName)
		prod.counters[streamName] = new(int64)
	}

	return conf.Errors.OrNil()
}

func (prod *Kinesis) bufferMessage(msg *core.Message) {
	prod.batch.AppendOrFlush(msg, prod.sendBatch, prod.IsActiveOrStopping, prod.Drop)
}

func (prod *Kinesis) sendBatchOnTimeOut() {
	// Flush if necessary
	if prod.batch.ReachedTimeThreshold(prod.flushFrequency) || prod.batch.ReachedSizeThreshold(prod.batch.Len()/2) {
		prod.sendBatch()
	}

	duration := time.Since(prod.lastMetricUpdate)
	prod.lastMetricUpdate = time.Now()

	for streamName, counter := range prod.counters {
		count := atomic.SwapInt64(counter, 0)

		tgo.Metric.Add(kinesisMetricMessages+streamName, count)
		tgo.Metric.SetF(kinesisMetricMessagesSec+streamName, float64(count)/duration.Seconds())
	}
}

func (prod *Kinesis) sendBatch() {
	prod.batch.Flush(prod.transformMessages)
}

func (prod *Kinesis) dropMessages(messages []*core.Message) {
	for _, msg := range messages {
		prod.Drop(msg)
	}
}

func (prod *Kinesis) transformMessages(messages []*core.Message) {
	streamRecords := make(map[core.MessageStreamID]*streamData)

	// Format and sort
	for idx, msg := range messages {
		currentMsg := *msg
		prod.ProducerBase.Format(&currentMsg)
		messageHash := fmt.Sprintf("%X-%d", currentMsg.StreamID, currentMsg.Sequence)

		// Fetch buffer for this stream
		records, recordsExists := streamRecords[currentMsg.StreamID]
		if !recordsExists {
			// Select the correct kinesis stream
			streamName, streamMapped := prod.streamMap[currentMsg.StreamID]
			if !streamMapped {
				streamName = core.StreamRegistry.GetStreamName(currentMsg.StreamID)
				prod.streamMap[currentMsg.StreamID] = streamName

				tgo.Metric.New(kinesisMetricMessages + streamName)
				tgo.Metric.New(kinesisMetricMessagesSec + streamName)
				prod.counters[streamName] = new(int64)
			}

			// Create buffers for this kinesis stream
			records = &streamData{
				content: &kinesis.PutRecordsInput{
					Records:    make([]*kinesis.PutRecordsRequestEntry, 0, len(messages)),
					StreamName: aws.String(streamName),
				},
				original: make([]*core.Message, 0, len(messages)),
			}
			streamRecords[currentMsg.StreamID] = records
		}

		// Append record to stream
		record := &kinesis.PutRecordsRequestEntry{
			Data:         currentMsg.Data,
			PartitionKey: aws.String(messageHash),
		}

		records.content.Records = append(records.content.Records, record)
		records.original = append(records.original, messages[idx])
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
			// Batch failed, drop all
			prod.Log.Error.Print("Kinesis write error: ", err)
			for _, msg := range records.original {
				prod.Drop(msg)
			}
		} else {
			// Check each message for errors
			for msgIdx, record := range result.Records {
				if record.ErrorMessage != nil {
					prod.Log.Error.Print("Kinesis message write error: ", *record.ErrorMessage)
					prod.Drop(records.original[msgIdx])
				}
			}
		}
	}
}

func (prod *Kinesis) close() {
	defer prod.WorkerDone()
	prod.CloseMessageChannel(prod.bufferMessage)
	prod.batch.Close(prod.transformMessages, prod.GetShutdownTimeout())
}

// Produce writes to stdout or stderr.
func (prod *Kinesis) Produce(workers *sync.WaitGroup) {
	prod.AddMainWorker(workers)

	prod.client = kinesis.New(session.New(prod.config))
	prod.TickerMessageControlLoop(prod.bufferMessage, prod.flushFrequency, prod.sendBatchOnTimeOut)
}
