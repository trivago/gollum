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
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/components"
)

const (
	kinesisOffsetNewest = "newest"
	kinesisOffsetOldest = "oldest"
)

// AwsKinesis consumer
//
// This consumer reads a message from an AWS Kinesis router.
//
// Parameters
//
// - KinesisStream: This value defines the stream to read from.
// By default this parameter is set to "default".
//
// - OffsetFile: This value defines a file to store the current offset per shard.
// To disable this parameter, set it to "". If the parameter is set and the file
// is found, consuming will start after the offset stored in the file.
// By default this parameter is set to "".
//
// - RecordsPerQuery: This value defines the number of records to pull per query.
// By default this parameter is set to "100".
//
// - RecordMessageDelimiter: This value defines the string to delimit messages
// within a record. To disable this parameter, set it to "".
// By default this parameter is set to "".
//
// - QuerySleepTimeMs: This value defines the number of milliseconds to sleep
// before trying to pull new records from a shard that did not return any records.
// By default this parameter is set to "1000".
//
// - RetrySleepTimeSec: This value defines the number of seconds to wait after
// trying to reconnect to a shard.
// By default this parameter is set to "4".
//
// - CheckNewShardsSec: This value sets a timer to update shards in Kinesis.
// You can set this parameter to "0" for disabling.
// By default this parameter is set to "0".
//
// - DefaultOffset: This value defines the message index to start reading from.
// Valid values are either "newest", "oldest", or a number.
// By default this parameter is set to "newest".
//
// Examples
//
// This example consumes a kinesis stream "myStream" and create messages:
//
//  KinesisIn:
//    Type: consumer.AwsKinesis
//    Credential:
//      Type: shared
//      File: /Users/<USERNAME>/.aws/credentials
//      Profile: default
//    Region: "eu-west-1"
//    KinesisStream: myStream
type AwsKinesis struct {
	core.SimpleConsumer `gollumdoc:"embed_type"`

	// AwsMultiClient is public to make AwsMultiClient.Configure() callable
	AwsMultiClient components.AwsMultiClient `gollumdoc:"embed_type"`

	stream          string        `config:"KinesisStream" default:"default"`
	offsetFile      string        `config:"OffsetFile"`
	recordsPerQuery int64         `config:"RecordsPerQuery" default:"100"`
	delimiter       []byte        `config:"RecordMessageDelimiter"`
	sleepTime       time.Duration `config:"QuerySleepTimeMs" default:"1000" metric:"ms"`
	//retryTime       time.Duration `config:"RetrySleepTimeSec" default:"4" metric:"sec"`
	shardTime time.Duration `config:"CheckNewShardsSec" default:"0" metric:"sec"`

	client        *kinesis.Kinesis
	offsets       map[string]string
	offsetType    string
	defaultOffset string
	running       bool
	offsetsGuard  *sync.RWMutex
}

func init() {
	core.TypeRegistry.Register(AwsKinesis{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *AwsKinesis) Configure(conf core.PluginConfigReader) {
	cons.offsets = make(map[string]string)
	cons.offsetsGuard = new(sync.RWMutex)

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

func (cons *AwsKinesis) marshalOffsets() ([]byte, error) {
	cons.offsetsGuard.RLock()
	defer cons.offsetsGuard.RUnlock()
	return json.Marshal(cons.offsets)
}

func (cons *AwsKinesis) storeOffsets() {
	if cons.offsetFile != "" {
		fileContents, err := cons.marshalOffsets()
		if err != nil {
			cons.Logger.Errorf("Failed to marshal kinesis offsets: %s", err.Error())
			return
		}

		if err := ioutil.WriteFile(cons.offsetFile, fileContents, 0644); err != nil {
			cons.Logger.Errorf("Failed to write kinesis offsets: %s", err.Error())
		}
	}
}

func (cons *AwsKinesis) createShardIteratorConfig(shardID string) *kinesis.GetRecordsInput {
	for cons.running {
		cons.offsetsGuard.RLock()
		offset := cons.offsets[shardID]
		cons.offsetsGuard.RUnlock()

		iteratorConfig := kinesis.GetShardIteratorInput{
			ShardId:                aws.String(shardID),
			ShardIteratorType:      aws.String(cons.offsetType),
			StreamName:             aws.String(cons.stream),
			StartingSequenceNumber: aws.String(offset),
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

func (cons *AwsKinesis) processShard(shardID string) {
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
					cons.Enqueue(msg)
				}
			} else {
				cons.Enqueue(record.Data)
			}

			// Why not an exclusive lock here?
			// - the map already has an entry for the shardID at this point.
			// - we are the only one writing to this field
			// - reading an offset is always a race as it can change at any time
			// - performance is crucial here
			// If we could be sure this actually IS a number, we could use atomic.Store here
			cons.offsetsGuard.RLock()
			cons.offsets[shardID] = *record.SequenceNumber
			cons.offsetsGuard.RUnlock()
		}

		cons.storeOffsets()
		recordConfig.ShardIterator = result.NextShardIterator
		time.Sleep(cons.sleepTime)
	}
}

func (cons *AwsKinesis) initKinesisClient() {
	sess, err := cons.AwsMultiClient.NewSessionWithOptions()
	if err != nil {
		cons.Logger.WithError(err).Error("Can't get proper aws config")
	}

	awsConfig := cons.AwsMultiClient.GetConfig()

	// set auto endpoint to s3 if setting is empty
	if awsConfig.Endpoint == nil || *awsConfig.Endpoint == "" {
		awsConfig.WithEndpoint(fmt.Sprintf("kinesis.%s.amazonaws.com", *awsConfig.Region))
	}

	cons.client = kinesis.New(sess, awsConfig)
}

func (cons *AwsKinesis) connect() error {
	cons.initKinesisClient()

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

		// No locks required here as connect is called only once on start
		if _, offsetStored := cons.offsets[*shard.ShardId]; !offsetStored {
			cons.offsets[*shard.ShardId] = cons.defaultOffset
		}
	}

	// No locks required here updateShards is not yet running and processShard
	// does not change the offsets structure.
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

func (cons *AwsKinesis) updateShards() {
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

			cons.offsetsGuard.Lock()
			if _, offsetStored := cons.offsets[*shard.ShardId]; !offsetStored {
				cons.offsets[*shard.ShardId] = cons.defaultOffset
				cons.Logger.Debugf("Starting kinesis consumer for %s:%s", cons.stream, *shard.ShardId)
				go cons.processShard(*shard.ShardId)
			}
			cons.offsetsGuard.Unlock()
		}
		time.Sleep(cons.shardTime)
	}
}

func (cons *AwsKinesis) close() {
	cons.running = false
	cons.WorkerDone()
}

// Consume listens to stdin.
func (cons *AwsKinesis) Consume(workers *sync.WaitGroup) {
	cons.AddMainWorker(workers)
	defer cons.close()

	if err := cons.connect(); err != nil {
		cons.Logger.Error("Connection error: ", err)
	} else {
		cons.ControlLoop()
	}
}
