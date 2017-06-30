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

package consumer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/trivago/gollum/core"
)

const (
	kinesisCredentialEnv    = "environment"
	kinesisCredentialStatic = "static"
	kinesisCredentialShared = "shared"
	kinesisCredentialNone   = "none"
	kinesisOffsetNewest     = "newest"
	kinesisOffsetOldest     = "oldest"
)

// Kinesis consumer plugin
//
// This consumer reads message from an AWS Kinesis router.
//
// Configuration example
//
//  - "consumer.Kinesis":
//    KinesisStream: "default"
//    Region: "eu-west-1"
//    Endpoint: "kinesis.eu-west-1.amazonaws.com"
//    DefaultOffset: "Newest"
//    OffsetFile: ""
//    RecordsPerQuery: 100
//    RecordMessageDelimiter: ""
//    QuerySleepTimeMs: 1000
//    RetrySleepTimeSec: 4
//    CredentialType: "none"
//    CredentialId: ""
//    CredentialToken: ""
//    CredentialSecret: ""
//    CredentialFile: ""
//    CredentialProfile: ""
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
// CredentialSecretm shared enables the parameters CredentialFile and
// CredentialProfile. None will not use any credentials and environment
// will pull the credentials from environmental settings.
// By default this is set to none.
//
// DefaultOffset defines the message index to start reading from.
// Valid values are either "Newest", "Oldest", or a number.
// The default value is "Newest".
//
// OffsetFile defines a file to store the current offset per shard.
// By default this is set to "", i.e. it is disabled.
// If a file is set and found consuming will start after the stored
// offset.
//
// RecordsPerQuery defines the number of records to pull per query.
// By default this is set to 100.
//
// RecordMessageDelimiter defines the string to delimit messages within a
// record. By default this is set to "", i.e. it is disabled.
//
// QuerySleepTimeMs defines the number of milliseconds to sleep before
// trying to pull new records from a shard that did not return any records.
// By default this is set to 1000.
//
// RetrySleepTimeSec defines the number of seconds to wait after trying to
// reconnect to a shard. By default this is set to 4.
type Kinesis struct {
	core.SimpleConsumer `gollumdoc:"embed_type"`
	client              *kinesis.Kinesis
	config              *aws.Config
	offsets             map[string]string
	offsetType          string
	defaultOffset       string
	stream              string        `config:"KinesisStream" default:"default"`
	offsetFile          string        `config:"OffsetFile"`
	recordsPerQuery     int64         `config:"RecordsPerQuery" default:"1000"`
	delimiter           []byte        `config:"RecordMessageDelimiter"`
	sleepTime           time.Duration `config:"QuerySleepTimeMs" default:"1000" metric:"ms"`
	retryTime           time.Duration `config:"RetrySleepTimeSec" default:"4" metric:"sec"`
	shardTime           time.Duration `config:"CheckNewShardsSec" default:"0" metric:"sec"`
	running             bool
}

func init() {
	core.TypeRegistry.Register(Kinesis{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *Kinesis) Configure(conf core.PluginConfigReader) {
	cons.offsets = make(map[string]string)

	// Config
	cons.config = aws.NewConfig()
	if endpoint := conf.GetString("Endpoint", "kinesis.eu-west-1.amazonaws.com"); endpoint != "" {
		cons.config.WithEndpoint(endpoint)
	}

	if region := conf.GetString("Region", "eu-west-1"); region != "" {
		cons.config.WithRegion(region)
	}

	// Credentials
	credentialType := strings.ToLower(conf.GetString("Credential/Type", kinesisCredentialNone))
	switch credentialType {
	case kinesisCredentialEnv:
		cons.config.WithCredentials(credentials.NewEnvCredentials())

	case kinesisCredentialStatic:
		id := conf.GetString("Credentia/lId", "")
		token := conf.GetString("Credential/Token", "")
		secret := conf.GetString("Credential/Secret", "")
		cons.config.WithCredentials(credentials.NewStaticCredentials(id, secret, token))

	case kinesisCredentialShared:
		filename := conf.GetString("Credential/File", "")
		profile := conf.GetString("Credential/Profile", "")
		cons.config.WithCredentials(credentials.NewSharedCredentials(filename, profile))

	case kinesisCredentialNone:
		// Nothing

	default:
		conf.Errors.Pushf("Unknown CredentialType: %s", credentialType)
		return
	}

	// Offset
	offsetValue := strings.ToLower(conf.GetString("DefaultOffset", kinesisOffsetNewest))
	switch offsetValue {
	case kinesisOffsetNewest:
		cons.offsetType = kinesis.ShardIteratorTypeLatest
		cons.defaultOffset = ""

	case kinesisOffsetOldest:
		cons.offsetType = kinesis.ShardIteratorTypeTrimHorizon
		cons.defaultOffset = ""

	default:
		cons.offsetType = kinesis.ShardIteratorTypeAtSequenceNumber
		_, err := strconv.ParseUint(offsetValue, 10, 64)
		if err != nil {
			cons.Logger.Errorf("Default offset must be \"%s\", \"%s\" or a number. %s given", kinesisOffsetNewest, kinesisOffsetOldest, offsetValue)
			offsetValue = "0"
		}
		cons.defaultOffset = offsetValue
	}

	if cons.offsetFile != "" {
		fileContents, err := ioutil.ReadFile(cons.offsetFile)
		if err != nil {
			cons.Logger.Errorf("Failed to open kinesis offset file: %s", err.Error())
		} else {
			cons.offsetType = kinesis.ShardIteratorTypeAfterSequenceNumber
			conf.Errors.Push(json.Unmarshal(fileContents, &cons.offsets))
		}
	}
}

func (cons *Kinesis) storeOffsets() {
	if cons.offsetFile != "" {
		fileContents, err := json.Marshal(cons.offsets)
		if err != nil {
			cons.Logger.Errorf("Failed to marshal kinesis offsets: %s", err.Error())
			return
		}

		if err := ioutil.WriteFile(cons.offsetFile, fileContents, 0644); err != nil {
			cons.Logger.Errorf("Failed to write kinesis offsets: %s", err.Error())
		}
	}
}

func (cons *Kinesis) createShardIteratorConfig(shardID string) *kinesis.GetRecordsInput {
	for cons.running {
		iteratorConfig := kinesis.GetShardIteratorInput{
			ShardId:                aws.String(shardID),
			ShardIteratorType:      aws.String(cons.offsetType),
			StreamName:             aws.String(cons.stream),
			StartingSequenceNumber: aws.String(cons.offsets[shardID]),
		}

		if *iteratorConfig.StartingSequenceNumber == "" {
			iteratorConfig.StartingSequenceNumber = nil
		} else {
			// starting sequence number requires ShardIteratorTypeAfterSequenceNumber
			iteratorConfig.ShardIteratorType = aws.String(kinesis.ShardIteratorTypeAfterSequenceNumber)
		}

		iterator, err := cons.client.GetShardIterator(&iteratorConfig)
		if err == nil && iterator.ShardIterator != nil {
			return &kinesis.GetRecordsInput{
				ShardIterator: iterator.ShardIterator,
				Limit:         aws.Int64(cons.recordsPerQuery),
			}
		}

		cons.Logger.Errorf("Failed to iterate shard %s:%s - %s", *iteratorConfig.StreamName, *iteratorConfig.ShardId, err.Error())
		time.Sleep(3 * time.Second)
	}

	return nil
}

func (cons *Kinesis) processShard(shardID string) {
	cons.AddWorker()
	defer cons.WorkerDone()
	recordConfig := (*kinesis.GetRecordsInput)(nil)

	for cons.running {
		if recordConfig == nil {
			recordConfig = cons.createShardIteratorConfig(shardID)
		}

		result, err := cons.client.GetRecords(recordConfig)
		if err != nil {
			cons.Logger.Errorf("Failed to get records from shard %s:%s - %s", cons.stream, shardID, err.Error())

			if AWSerr, isAWSerr := err.(awserr.Error); isAWSerr {
				switch AWSerr.Code() {
				case "ProvisionedThroughputExceededException":
					// We reached thethroughput limit
					time.Sleep(5 * time.Second)

				case "ExpiredIteratorException":
					// We need to create a new iterator
					recordConfig = nil
				}
			}
			continue // ### continue ###
		}

		if result.NextShardIterator == nil {
			cons.Logger.Warningf("Shard %s:%s has been closed", cons.stream, shardID)
			return // ### return, closed ###
		}

		for _, record := range result.Records {
			if record == nil {
				continue // ### continue ###
			}

			if len(cons.delimiter) > 0 {
				messages := bytes.Split(record.Data, cons.delimiter)
				for _, msg := range messages {
					cons.Enqueue([]byte(msg))
				}
			} else {
				cons.Enqueue(record.Data)
			}
			cons.offsets[shardID] = *record.SequenceNumber
		}

		cons.storeOffsets()
		recordConfig.ShardIterator = result.NextShardIterator
		time.Sleep(cons.sleepTime)
	}
}

func (cons *Kinesis) connect() error {
	cons.client = kinesis.New(session.New(cons.config))

	// Get shard ids for stream
	streamQuery := &kinesis.DescribeStreamInput{
		StreamName: aws.String(cons.stream),
	}

	streamInfo, err := cons.client.DescribeStream(streamQuery)
	if err != nil {
		return err
	}

	if streamInfo.StreamDescription == nil {
		return fmt.Errorf("StreamDescription could not be retrieved")
	}

	cons.running = true
	for _, shard := range streamInfo.StreamDescription.Shards {
		if shard.ShardId == nil {
			return fmt.Errorf("ShardId could not be retrieved")
		}

		if _, offsetStored := cons.offsets[*shard.ShardId]; !offsetStored {
			cons.offsets[*shard.ShardId] = cons.defaultOffset
		}
	}

	for shardID := range cons.offsets {
		cons.Logger.Debugf("Starting consumer for %s:%s", cons.stream, shardID)
		go cons.processShard(shardID)
	}

	if cons.shardTime > 0 {
		cons.AddWorker()
		time.AfterFunc(cons.shardTime, cons.updateShards)
	}

	return nil
}

func (cons *Kinesis) updateShards() {
	defer cons.WorkerDone()

	streamQuery := &kinesis.DescribeStreamInput{
		StreamName: aws.String(cons.stream),
	}

	for cons.running {
		streamInfo, err := cons.client.DescribeStream(streamQuery)
		if err != nil {
			cons.Logger.Warningf("StreamInfo could not be retrieved.")
		}

		if streamInfo.StreamDescription == nil {
			cons.Logger.Warningf("StreamDescription could not be retrieved.")
			continue
		}

		cons.running = true
		for _, shard := range streamInfo.StreamDescription.Shards {
			if shard.ShardId == nil {
				cons.Logger.Warningf("ShardId could not be retrieved.")
				continue
			}

			if _, offsetStored := cons.offsets[*shard.ShardId]; !offsetStored {
				cons.offsets[*shard.ShardId] = cons.defaultOffset
				cons.Logger.Debugf("Starting kinesis consumer for %s:%s", cons.stream, *shard.ShardId)
				go cons.processShard(*shard.ShardId)
			}
		}
		time.Sleep(cons.shardTime)
	}
}

func (cons *Kinesis) close() {
	cons.running = false
	cons.WorkerDone()
}

// Consume listens to stdin.
func (cons *Kinesis) Consume(workers *sync.WaitGroup) {
	cons.AddMainWorker(workers)
	defer cons.close()

	if err := cons.connect(); err != nil {
		cons.Logger.Error("Connection error: ", err)
	} else {
		cons.ControlLoop()
	}
}
