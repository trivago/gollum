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
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	s3CredentialEnv    = "environment"
	s3CredentialStatic = "static"
	s3CredentialShared = "shared"
	s3CredentialNone   = "none"
)

// S3 producer plugin
// This producer sends data to an AWS S3 Bucket.
// Configuration example
//
//  - "producer.S3":
//    Region: "eu-west-1"
//    Endpoint: "s3-eu-west-1.amazonaws.com"
//    StorageClass: "STANDARD"
//    CredentialType: "none"
//    CredentialId: ""
//    CredentialToken: ""
//    CredentialSecret: ""
//    CredentialFile: ""
//    CredentialProfile: ""
//    BatchMaxMessages: 5000
//    ObjectMaxMessages: 5000
//    ObjectMessageDelimiter: "\n"
//    SendTimeframeMs: 10000
//    BatchTimeoutSec: 30
//    TimestampWrite: "2006-01-02T15:04:05"
//    StreamMapping:
//      "*" : "default"
//
// Region defines the amazon region of your s3 bucket.
// By default this is set to "eu-west-1".
//
// Endpoint defines the amazon endpoint for your s3 bucket.
// By default this is set to "s3-eu-west-1.amazonaws.com"
//
// Endpoint defines the amazon s3 storage class for objects created.
// By default this is set to "STANDARD"
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
// BatchMaxMessages defines the maximum number of messages to upload per
// batch. By default this is set to 5000.
//
// ObjectMaxMessages defines the maximum number of messages to join into
// an s3 object. By default this is set to 5000.
//
// ObjectMessageDelimiter defines the string to delimit messages within
// an s3 object. By default this is set to "\n".
//
// SendTimeframeMs defines the timeframe in milliseconds in which a second
// batch send can be triggered. By default this is set to 10000, i.e. ten
// upload operations per second per s3 path.
//
// BatchTimeoutSec defines the number of seconds after which a batch is
// flushed automatically. By default this is set to 30.
//
// TimestampWrite defines the go timestamp format that will be used in naming
// objects. Objects are named <s3_path><timestamp><sha1>. By default timestamp
// is set to "2006-01-02T15:04:05".
//
// StreamMapping defines a translation from gollum stream to s3 path. If
// no mapping is given the gollum stream name is used as s3 path.
type S3 struct {
	core.ProducerBase
	client            *s3.S3
	config            *aws.Config
	storageClass      string
	streamMap         map[core.MessageStreamID]string
	batch             core.MessageBatch
	objectMaxMessages int
	delimiter         []byte
	flushFrequency    time.Duration
	timeWrite    string
	lastSendTime      time.Time
	sendTimeLimit     time.Duration
	counters          map[string]*int64
	lastMetricUpdate  time.Time
}

const (
	s3MetricMessages    = "S3:Messages-"
	s3MetricMessagesSec = "S3:MessagesSec-"
)

type objectData struct {
	objects            [][]byte
	original           [][]*core.Message
	s3Bucket           string
	s3Path             string
	s3Prefix           string
	lastObjectMessages int
}

func init() {
	shared.TypeRegistry.Register(S3{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *S3) Configure(conf core.PluginConfig) error {
	err := prod.ProducerBase.Configure(conf)
	if err != nil {
		return err
	}
	prod.SetStopCallback(prod.close)

	prod.storageClass = conf.GetString("StorageClass", "STANDARD")
	prod.streamMap = conf.GetStreamMap("StreamMapping", "")
	prod.batch = core.NewMessageBatch(conf.GetInt("BatchMaxMessages", 5000))
	prod.objectMaxMessages = conf.GetInt("ObjectMaxMessages", 5000)
	prod.delimiter = []byte(conf.GetString("ObjectMessageDelimiter", "\n"))
	prod.flushFrequency = time.Duration(conf.GetInt("BatchTimeoutSec", 30)) * time.Second
	prod.sendTimeLimit = time.Duration(conf.GetInt("SendTimeframeMs", 10000)) * time.Millisecond
	prod.timeWrite = conf.GetString("TimestampWrite", "2006-01-02T15:04:05")
	prod.lastSendTime = time.Now()
	prod.counters = make(map[string]*int64)
	prod.lastMetricUpdate = time.Now()

	if prod.objectMaxMessages < 1 {
		prod.objectMaxMessages = 1
		Log.Warning.Print("ObjectMaxMessages was < 1. Defaulting to 1.")
	}

	if prod.objectMaxMessages > 1 && len(prod.delimiter) == 0 {
		prod.delimiter = []byte("\n")
		Log.Warning.Print("ObjectMessageDelimiter was empty. Defaulting to \"\\n\".")
	}

	// Config
	defaultEndpoint := "s3.amazonaws.com"
	prod.config = aws.NewConfig()
	if region := conf.GetString("Region", "eu-west-1"); region != "" {
		prod.config.WithRegion(region)
		if region != "us-east-1" {
			defaultEndpoint = "s3-" + region + ".amazonaws.com"
		}
	}

	if endpoint := conf.GetString("Endpoint", defaultEndpoint); endpoint != "" {
		prod.config.WithEndpoint(endpoint)
	}

	// Credentials
	credentialType := strings.ToLower(conf.GetString("CredentialType", s3CredentialNone))
	switch credentialType {
	case s3CredentialEnv:
		prod.config.WithCredentials(credentials.NewEnvCredentials())

	case s3CredentialStatic:
		id := conf.GetString("CredentialId", "")
		token := conf.GetString("CredentialToken", "")
		secret := conf.GetString("CredentialSecret", "")
		prod.config.WithCredentials(credentials.NewStaticCredentials(id, secret, token))

	case s3CredentialShared:
		filename := conf.GetString("CredentialFile", "")
		profile := conf.GetString("CredentialProfile", "")
		prod.config.WithCredentials(credentials.NewSharedCredentials(filename, profile))

	case s3CredentialNone:
		// Nothing

	default:
		return fmt.Errorf("Unknown CredentialType: %s", credentialType)
	}

	for _, s3Path := range prod.streamMap {
		shared.Metric.New(s3MetricMessages + s3Path)
		shared.Metric.New(s3MetricMessagesSec + s3Path)
		prod.counters[s3Path] = new(int64)
	}

	return nil
}

func (prod *S3) bufferMessage(msg core.Message) {
	prod.batch.AppendOrFlush(msg, prod.sendBatch, prod.IsActiveOrStopping, prod.Drop)
}

func (prod *S3) sendBatchOnTimeOut() {
	// Flush if necessary
	if prod.batch.ReachedTimeThreshold(prod.flushFrequency) || prod.batch.ReachedSizeThreshold(prod.batch.Len()/2) {
		prod.sendBatch()
	}

	duration := time.Since(prod.lastMetricUpdate)
	prod.lastMetricUpdate = time.Now()

	for s3Path, counter := range prod.counters {
		count := atomic.SwapInt64(counter, 0)

		shared.Metric.Add(s3MetricMessages+s3Path, count)
		shared.Metric.SetF(s3MetricMessagesSec+s3Path, float64(count)/duration.Seconds())
	}
}

func (prod *S3) sendBatch() {
	prod.batch.Flush(prod.transformMessages)
}

func (prod *S3) dropMessages(messages []core.Message) {
	for _, msg := range messages {
		prod.Drop(msg)
	}
}

func (prod *S3) transformMessages(messages []core.Message) {
	streamObjects := make(map[core.MessageStreamID]*objectData)

	// Format and sort
	for idx, msg := range messages {
		msgData, streamID := prod.ProducerBase.Format(msg)

		// Fetch buffer for this stream
		objects, objectsExists := streamObjects[streamID]
		if !objectsExists {
			// Select the correct s3 path
			s3Path, streamMapped := prod.streamMap[streamID]
			if !streamMapped {
				s3Path, streamMapped = prod.streamMap[core.WildcardStreamID]
				if !streamMapped {
					s3Path = core.StreamRegistry.GetStreamName(streamID)
					prod.streamMap[streamID] = s3Path

					shared.Metric.New(s3MetricMessages + s3Path)
					shared.Metric.New(s3MetricMessagesSec + s3Path)
					prod.counters[s3Path] = new(int64)
				}
			}

			// split bucket from prefix in path
			s3Bucket, s3Prefix := s3Path, ""
			if strings.Contains(s3Path, "/") {
				split := strings.SplitN(s3Path, "/", 2)
				s3Bucket, s3Prefix = split[0], split[1]
			}

			// Create buffers for this s3 path
			maxLength := len(messages) / prod.objectMaxMessages + 1
			objects = &objectData{
			    objects:            make([][]byte, 0, maxLength),
				s3Bucket:           s3Bucket,
				s3Path:				s3Path,
				s3Prefix:           s3Prefix,
				original:           make([][]*core.Message, 0, maxLength),
				lastObjectMessages: 0,
			}
			streamObjects[streamID] = objects
		}

		// Fetch object for this buffer
		objectExists := len(objects.objects) > 0
		if !objectExists || objects.lastObjectMessages + 1 > prod.objectMaxMessages {
			// Append object to stream
			objects.objects = append(objects.objects, make([]byte, 0, len(msgData)))
			objects.original = append(objects.original, make([]*core.Message, 0, prod.objectMaxMessages))
			objects.lastObjectMessages = 0
		} else {
			objects.objects[len(objects.objects)-1] = append(objects.objects[len(objects.objects)-1], prod.delimiter...)
		}

		// Append message to object
		objects.objects[len(objects.objects)-1] = append(objects.objects[len(objects.objects)-1], msgData...)
		objects.lastObjectMessages += 1
		objects.original[len(objects.original)-1] = append(objects.original[len(objects.original)-1], &messages[idx])
	}

	sleepDuration := prod.sendTimeLimit - time.Since(prod.lastSendTime)
	if sleepDuration > 0 {
		time.Sleep(sleepDuration)
	}


	// Send to S3
	for _, objects := range streamObjects {
		for idx, object := range objects.objects {
			hash := sha1.Sum(object)
			key := objects.s3Prefix + time.Now().Format(prod.timeWrite) + hex.EncodeToString(hash[:])
			params := &s3.PutObjectInput{
				Bucket:       aws.String(objects.s3Bucket),
				Key:          aws.String(key),
				Body:         bytes.NewReader(object),
				StorageClass: aws.String(prod.storageClass),
			}
			_, err := prod.client.PutObject(params)
			atomic.AddInt64(prod.counters[objects.s3Path], int64(1))
			if err != nil {
				Log.Error.Print("S3 write error: ", err)
				for _, msg := range objects.original[idx] {
					prod.Drop(*msg)
				}
			}
		}
	}
}

func (prod *S3) close() {
	defer prod.WorkerDone()
	prod.CloseMessageChannel(prod.bufferMessage)
	prod.batch.Close(prod.transformMessages, prod.GetShutdownTimeout())
}

func (prod *S3) Produce(workers *sync.WaitGroup) {
	prod.AddMainWorker(workers)

	prod.client = s3.New(session.New(prod.config))
	prod.TickerMessageControlLoop(prod.bufferMessage, prod.flushFrequency, prod.sendBatchOnTimeOut)
}
