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
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
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
	core.ProducerBase
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
}

const (
	kinesisMetricMessages    = "Kinesis:Messages-"
	kinesisMetricMessagesSec = "Kinesis:MessagesSec-"
)

type streamData struct {
	content            []*kinesis.PutRecordsInput
	original           [][][]*core.Message
	streamName         string
	lastRecordMessages int
	lastRequestSize    int
	recordMaxMessages  int
	recordMaxSize      int
	requestMaxSize     int
	requestMaxRecords  int
}

func newStreamData(streamName string, recordMaxMessages int) *streamData {
	return &streamData{
		content: make([]*kinesis.PutRecordsInput, 0, 1),
		original: make([][][]*core.Message, 0, 1),
		streamName: streamName,
		lastRecordMessages: 0,
		lastRequestSize: 0,
		recordMaxMessages: recordMaxMessages,
		recordMaxSize: 1<<20, // 1 MB per record is the api limit
		requestMaxSize: 5<<20, // 5 MB per request is the api limit
		requestMaxRecords: 500, // 500 records per request is the api limit
	}
}

func (sd *streamData) AddPutRecordsInput() *kinesis.PutRecordsInput {
	input := &kinesis.PutRecordsInput{
		Records:    make([]*kinesis.PutRecordsRequestEntry, 0, sd.requestMaxRecords),
		StreamName: aws.String(sd.streamName),
	}
	sd.content = append(sd.content, input)
	sd.original = append(sd.original, make([][]*core.Message, 0, sd.requestMaxRecords))
	sd.lastRequestSize = 0
	return input
}

func (sd *streamData) GetPutRecordsInput() (*kinesis.PutRecordsInput) {
	if len(sd.content) == 0 {
		return sd.AddPutRecordsInput()
	} else {
		return sd.content[len(sd.content)-1]
	}
}

func (sd *streamData) AddRecord() *kinesis.PutRecordsRequestEntry {
	input := sd.GetPutRecordsInput()
	record := &kinesis.PutRecordsRequestEntry{
		Data: make([]byte, 0, 0),
	}
	if len(input.Records) + 1 > sd.requestMaxRecords {
		input = sd.AddPutRecordsInput()
	}
	input.Records = append(input.Records, record)
	sd.original[len(sd.original)-1] = append(sd.original[len(sd.original)-1], make([]*core.Message, 0, sd.recordMaxMessages))
	sd.lastRecordMessages = 0
	return record
}

func (sd *streamData) GetRecord() *kinesis.PutRecordsRequestEntry {
	input := sd.GetPutRecordsInput()
	if len(input.Records) == 0 {
		return sd.AddRecord()
	} else {
		return input.Records[len(input.Records)-1]
	}
}

func (sd *streamData) AddMessage(delimiter []byte, data []byte, streamID core.MessageStreamID, msg *core.Message) {
	record := sd.GetRecord()
	if len(record.Data) > 0 {
		addSize := len(delimiter) + len(data)
		createRecord := false
		if addSize + len(record.Data) > sd.recordMaxSize || sd.lastRecordMessages + 1 > sd.recordMaxMessages {
			createRecord = true
			addSize = len(data)
		}
		if addSize + sd.lastRequestSize > sd.requestMaxSize {
			sd.AddPutRecordsInput()
			record = sd.GetRecord()
		} else if createRecord {
			record = sd.AddRecord()
		}
	}
	if len(record.Data) > 0 {
		record.Data = append(record.Data, delimiter...)
		sd.lastRequestSize = sd.lastRequestSize + len(delimiter)
	} else {
		record.PartitionKey = aws.String(fmt.Sprintf("%X-%d", streamID, msg.Sequence))
	}
	record.Data = append(record.Data, data...)
	sd.lastRequestSize = sd.lastRequestSize + len(data)
	requests := len(sd.original)
	records := len(sd.original[requests-1])
	sd.original[requests-1][records-1] = append(sd.original[requests-1][records-1], msg)
	sd.lastRecordMessages++
}

func init() {
	shared.TypeRegistry.Register(Kinesis{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *Kinesis) Configure(conf core.PluginConfig) error {
	err := prod.ProducerBase.Configure(conf)
	if err != nil {
		return err
	}
	prod.SetStopCallback(prod.close)

	prod.streamMap = conf.GetStreamMap("StreamMapping", "")
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
		Log.Warning.Print("RecordMaxMessages was < 1. Defaulting to 1.")
	}

	if prod.recordMaxMessages > 1 && len(prod.delimiter) == 0 {
		prod.delimiter = []byte("\n")
		Log.Warning.Print("RecordMessageDelimiter was empty. Defaulting to \"\\n\".")
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
		shared.Metric.New(kinesisMetricMessages + streamName)
		shared.Metric.New(kinesisMetricMessagesSec + streamName)
		prod.counters[streamName] = new(int64)
	}

	return nil
}

func (prod *Kinesis) bufferMessage(msg core.Message) {
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

		shared.Metric.Add(kinesisMetricMessages+streamName, count)
		shared.Metric.SetF(kinesisMetricMessagesSec+streamName, float64(count)/duration.Seconds())
	}
}

func (prod *Kinesis) sendBatch() {
	prod.batch.Flush(prod.transformMessages)
}

func (prod *Kinesis) dropMessages(messages []core.Message) {
	for _, msg := range messages {
		prod.Drop(msg)
	}
}

func (prod *Kinesis) transformMessages(messages []core.Message) {
	streamRecords := make(map[core.MessageStreamID]*streamData)

	// Format and sort
	for idx, msg := range messages {
		msgData, streamID := prod.ProducerBase.Format(msg)

		// Fetch buffer for this stream
		requests, requestsExists := streamRecords[streamID]
		if !requestsExists {
			// Select the correct kinesis stream
			streamName, streamMapped := prod.streamMap[streamID]
			if !streamMapped {
				streamName, streamMapped = prod.streamMap[core.WildcardStreamID]
				if !streamMapped {
					streamName = core.StreamRegistry.GetStreamName(streamID)
					prod.streamMap[streamID] = streamName

					shared.Metric.New(kinesisMetricMessages + streamName)
					shared.Metric.New(kinesisMetricMessagesSec + streamName)
					prod.counters[streamName] = new(int64)
				}
			}
			requests = newStreamData(streamName, prod.recordMaxMessages)
			streamRecords[streamID] = requests
		}

		requests.AddMessage(prod.delimiter, msgData, streamID, &messages[idx])
	}

	sleepDuration := prod.sendTimeLimit - time.Since(prod.lastSendTime)
	if sleepDuration > 0 {
		time.Sleep(sleepDuration)
	}

	// Send to Kinesis
	for _, requests := range streamRecords {
		for reqIdx, request := range requests.content {
			result, err := prod.client.PutRecords(request)
			atomic.AddInt64(prod.counters[requests.streamName], int64(len(request.Records)))

			if err != nil {
				// request failed, drop all messages in request
				Log.Error.Print("Kinesis write error: ", err)
				for _, messages := range requests.original[reqIdx] {
					for _, msg := range messages {
						prod.Drop(*msg)
					}
				}
			} else {
				// Check each record for errors
				for recIdx, record := range result.Records {
					if record.ErrorMessage != nil {
						// record failed, drop all messages in record
						Log.Error.Print("Kinesis record write error: ", *record.ErrorMessage)
						for _, msg := range requests.original[reqIdx][recIdx] {
							prod.Drop(*msg)
						}
					}
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
