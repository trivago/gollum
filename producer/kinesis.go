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
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	"strings"
	"sync"
	"time"
)

const (
	kinesisCredentialEnv    = "environment"
	kinesisCredentialStatic = "static"
	kinesisCredentialShared = "shared"
	kinesisCredentialNone   = "none"
)

// Kinesis producer plugin
type Kinesis struct {
	core.ProducerBase
	client         *kinesis.Kinesis
	config         *aws.Config
	streamMap      map[core.MessageStreamID]string
	batch          core.MessageBatch
	flushFrequency time.Duration
	timeout        time.Duration
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

	prod.streamMap = conf.GetStreamMap("StreamMapping", "")
	prod.batch = core.NewMessageBatch(conf.GetInt("BatchMaxMessages", 500))
	prod.flushFrequency = time.Duration(conf.GetInt("BatchTimeoutSec", 3)) * time.Second
	prod.timeout = time.Duration(conf.GetInt("TimoutMs", 1500)) * time.Millisecond

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
		return fmt.Errorf("Unknwon CredentialType: %s", credentialType)
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
}

func (prod *Kinesis) sendBatch() {
	prod.batch.Flush(prod.transformMessages)
}

func (prod *Kinesis) dropMessages(messages []core.Message) {
	for _, msg := range messages {
		prod.Drop(msg)
	}
}

type inputData struct {
	data     kinesis.PutRecordsInput
	original []*core.Message
}

func (prod *Kinesis) transformMessages(messages []core.Message) {
	streamRecords := make(map[core.MessageStreamID]inputData)

	// Format and sort
	for _, msg := range messages {
		msgData, streamID := prod.ProducerBase.Format(msg)

		// Fetch buffer for this stream
		recordsInput, recordsExists := streamRecords[streamID]
		if !recordsExists {
			// Select the correct kinesis stream
			streamName, streamMapped := prod.streamMap[streamID]
			if !streamMapped {
				streamName = core.StreamRegistry.GetStreamName(streamID)
			}

			// Create buffers for this kinesis stream
			recordsInput = inputData{
				data: kinesis.PutRecordsInput{
					Records:    make([]*kinesis.PutRecordsRequestEntry, 0, len(messages)),
					StreamName: aws.String(streamName),
				},
				original: make([]*core.Message, 0, len(messages)),
			}
			streamRecords[streamID] = recordsInput
		}

		// Append record to stream
		record := &kinesis.PutRecordsRequestEntry{
			Data:         msgData,
			PartitionKey: aws.String(fmt.Sprintf("%X-%d", streamID, msg.Sequence)),
		}
		recordsInput.data.Records = append(recordsInput.data.Records, record)
		recordsInput.original = append(recordsInput.original, &msg)
	}

	// Send to Kinesis
	for _, records := range streamRecords {
		result, err := prod.client.PutRecords(&records.data)
		if err != nil {
			// Batch failed, drop all
			Log.Error.Print("Kinesis write error: ", err)
			for _, msg := range records.original {
				prod.Drop(*msg)
			}
		} else {
			// Check each message for errors
			for msgIdx, record := range result.Records {
				if record.ErrorMessage == nil {
					Log.Error.Print("Kinesis message write error: ", *record.ErrorMessage)
					prod.Drop(*records.original[msgIdx])
				}
			}
		}
	}

	time.Sleep(500)
}

// Produce writes to stdout or stderr.
func (prod *Kinesis) Produce(workers *sync.WaitGroup) {
	defer prod.WorkerDone()
	prod.AddMainWorker(workers)

	prod.client = kinesis.New(session.New(prod.config))
	prod.TickerMessageControlLoop(prod.bufferMessage, prod.timeout, prod.sendBatchOnTimeOut)
}
