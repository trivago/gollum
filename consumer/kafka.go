// Copyright 2015 trivago GmbH
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
	"encoding/json"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	kafka "gopkg.in/Shopify/sarama.v1"
	"io/ioutil"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	kafkaOffsetNewset = "newest"
	kafkaOffsetOldest = "oldest"
)

// Kafka consumer plugin
// Configuration example
//
//   - "consumer.Kafka":
//     Enable: true
//     DefaultOffset: "Newest"
//     OffsetFile: "/tmp/gollum_kafka.idx"
//     ClientID: "logger"
//     MaxOpenRequests: 6
//     ServerTimeoutSec: 10
//     MaxFetchSizeByte: 8192
//     MinFetchSizeByte: 0
//     FetchTimeoutMs: 500
//     MessageBufferCount: 1024
//     PresistTimoutMs: 1000
//     ElectRetries: 5
//     ElectTimeoutMs: 300
//     MetadataRefreshMs: 3000
//     Servers:
//       - "192.168.222.30:9092"
//       - "192.168.222.31:9092"
//
// The kafka consumer reads from a given kafka topic. This consumer is based on
// the sarama library so most settings relate to the settings from this library.
// This consumer can be paused.
//
// DefaultOffset defines the message index to start reading from.
// Valid values are either "Newset", "Oldest", or a number.
// The default value is "Newest".
//
// OffsetFile defines a path to a file containing the current index per topic
// partition. If a file is given the index stored in this file will be used as
// the default offset for a stored parition. If the partition is not stored in
// this file DefaultOffset is used.
//
// ClientId sets the client id of this producer. By default this is "gollum".
//
// MaxOpenRequests defines the number of simultanious connections are allowed.
// By default this is set to 5.
//
// ServerTimeoutSec defines the time after which a connection is set to timed
// out. By default this is set to 30 seconds.
//
// MaxFetchSizeByte sets the maximum size of a message to fetch. Larger messages
// will be ignored. By default this is set to 0 (fetch all messages).
//
// MinFetchSizeByte defines the minimum amout of data to fetch from Kafka per
// request. If less data is available the broker will wait. By default this is
// set to 1.
//
// FetchTimeoutMs defines the time in milliseconds the broker will wait for
// MinFetchSizeByte to be reached before processing data anyway. By default this
// is set to 250ms.
//
// MessageBufferCount sets the internal channel size for the kafka client.
// By default this is set to 256.
//
// PresistTimoutMs defines the time in milliseconds between writes to OffsetFile.
// By default this is set to 5000. Shorter durations reduce the amount of
// duplicate messages after a fail but increases I/O.
//
// ElectRetries defines how many times to retry during a leader election.
// By default this is set to 3.
//
// ElectTimeoutMs defines the number of milliseconds to wait for the cluster to
// elect a new leader. Defaults to 250.
//
// MetadataRefreshMs set the interval in seconds for fetching cluster metadata.
// By default this is set to 10000. This corresponds to the JVM setting
// `topic.metadata.refresh.interval.ms`.
//
// Servers contains the list of all kafka servers to connect to. This setting
// is mandatory and thus has no defaults.
type Kafka struct {
	core.ConsumerBase
	servers        []string
	topic          string
	client         kafka.Client
	config         *kafka.Config
	consumer       kafka.Consumer
	offsetFile     string
	defaultOffset  int64
	offsets        map[int32]int64
	MaxPartitionID int32
	persistTimeout time.Duration
}

func init() {
	shared.RuntimeType.Register(Kafka{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *Kafka) Configure(conf core.PluginConfig) error {
	err := cons.ConsumerBase.Configure(conf)
	if err != nil {
		return err
	}

	if !conf.HasValue("Servers") {
		return core.NewConsumerError("No servers configured for consumer.Kafka")
	}

	cons.servers = conf.GetStringArray("Servers", []string{})
	cons.topic = conf.GetString("Topic", "default")
	cons.offsetFile = conf.GetString("OffsetFile", "")
	cons.persistTimeout = time.Duration(conf.GetInt("PresistTimoutMs", 5000)) * time.Millisecond
	cons.offsets = make(map[int32]int64)
	cons.MaxPartitionID = 0

	cons.config = kafka.NewConfig()
	cons.config.ClientID = conf.GetString("ClientId", "gollum")
	cons.config.ChannelBufferSize = conf.GetInt("MessageBufferCount", 256)

	cons.config.Net.MaxOpenRequests = conf.GetInt("MaxOpenRequests", 5)
	cons.config.Net.DialTimeout = time.Duration(conf.GetInt("ServerTimeoutSec", 30)) * time.Second
	cons.config.Net.ReadTimeout = cons.config.Net.DialTimeout
	cons.config.Net.WriteTimeout = cons.config.Net.DialTimeout

	cons.config.Metadata.Retry.Max = conf.GetInt("ElectRetries", 3)
	cons.config.Metadata.Retry.Backoff = time.Duration(conf.GetInt("ElectTimeoutMs", 250)) * time.Millisecond
	cons.config.Metadata.RefreshFrequency = time.Duration(conf.GetInt("MetadataRefreshMs", 10000)) * time.Millisecond

	cons.config.Consumer.Fetch.Min = int32(conf.GetInt("MinFetchSizeByte", 1))
	cons.config.Consumer.Fetch.Max = int32(conf.GetInt("MaxFetchSizeByte", 0))
	cons.config.Consumer.Fetch.Default = int32(conf.GetInt("MaxFetchSizeByte", 32768))
	cons.config.Consumer.MaxWaitTime = time.Duration(conf.GetInt("FetchTimeoutMs", 250)) * time.Millisecond

	offsetValue := strings.ToLower(conf.GetString("DefaultOffset", kafkaOffsetNewset))
	switch offsetValue {
	case kafkaOffsetNewset:
		cons.defaultOffset = kafka.OffsetNewest

	case kafkaOffsetOldest:
		cons.defaultOffset = kafka.OffsetOldest

	default:
		cons.defaultOffset, _ = strconv.ParseInt(offsetValue, 10, 64)
		fileContents, err := ioutil.ReadFile(cons.offsetFile)
		if err != nil {
			return err
		}

		// Decode the JSON file into the partition -> offset map
		encodedOffsets := make(map[string]int64)
		err = json.Unmarshal(fileContents, &encodedOffsets)
		if err != nil {
			return err
		}

		for k, v := range encodedOffsets {
			id, err := strconv.Atoi(k)
			if err != nil {
				return err
			}
			cons.offsets[int32(id)] = v
		}
	}

	return nil
}

// Restart the consumer after wating for persistTimeout
func (cons *Kafka) retry(partitionID int32, err error) {
	Log.Error.Printf("Restarting kafka consumer (%s:%d) - %s", cons.topic, cons.offsets[partitionID], err.Error())
	time.Sleep(cons.persistTimeout)

	cons.readFromPartition(partitionID)
}

// Main fetch loop for kafka events
func (cons *Kafka) readFromPartition(partitionID int32) {
	partCons, err := cons.consumer.ConsumePartition(cons.topic, partitionID, cons.offsets[partitionID])
	if err != nil {
		if !cons.client.Closed() {
			go func() {
				defer shared.RecoverShutdown()
				cons.retry(partitionID, err)
			}()
		}
		return // ### return, stop this consumer ###
	}

	// Make sure we wait for all consumers to end

	cons.AddWorker()
	defer func() {
		if !cons.client.Closed() {
			partCons.Close()
		}
		cons.WorkerDone()
	}()

	// Loop over worker

	for !cons.client.Closed() {
		if cons.IsPaused() {
			runtime.Gosched()
			continue
		}

		select {
		case event := <-partCons.Messages():
			cons.offsets[partitionID] = event.Offset

			// Offset is always relative to the partition, so we create "buckets"
			// i.e. we are able to reconstruct partiton and local offset from
			// the sequence number.
			//
			// To generate this we use:
			// seq = offset * numPartition + partitionId
			//
			// Reading can be done via:
			// seq % numPartition = partitionId
			// seq / numPartition = offset

			sequence := uint64(event.Offset*int64(cons.MaxPartitionID) + int64(partitionID))
			cons.Enqueue(event.Value, sequence)

		case err := <-partCons.Errors():
			Log.Error.Print("Kafka consumer error:", err)

		default:
			runtime.Gosched()
		}
	}
}

// Start one consumer per partition as a go routine
func (cons *Kafka) startConsumers() error {
	var err error

	cons.client, err = kafka.NewClient(cons.servers, cons.config)
	if err != nil {
		return err
	}

	cons.consumer, err = kafka.NewConsumerFromClient(cons.client)
	if err != nil {
		return err
	}

	partitions, err := cons.client.Partitions(cons.topic)
	if err != nil {
		return err
	}

	for _, partition := range partitions {
		if _, mapped := cons.offsets[partition]; !mapped {
			cons.offsets[partition] = cons.defaultOffset
		}
		if partition > cons.MaxPartitionID {
			cons.MaxPartitionID = partition
		}
	}

	for _, partition := range partitions {
		partition := partition
		if _, mapped := cons.offsets[partition]; !mapped {
			cons.offsets[partition] = cons.defaultOffset
		}

		go func() {
			defer shared.RecoverShutdown()
			cons.readFromPartition(partition)
		}()
	}

	return nil
}

// Write index file to disc
func (cons *Kafka) dumpIndex() {
	if cons.offsetFile != "" {
		encodedOffsets := make(map[string]int64)
		for k, v := range cons.offsets {
			encodedOffsets[strconv.Itoa(int(k))] = v
		}

		data, err := json.Marshal(encodedOffsets)
		if err != nil {
			Log.Error.Print("Kafka index file write error - ", err)
		} else {
			ioutil.WriteFile(cons.offsetFile, data, 0644)
		}
	}
}

// Consume starts a kafka consumer per partition for this topic
func (cons *Kafka) Consume(workers *sync.WaitGroup) {
	cons.SetWorkerWaitGroup(workers)

	err := cons.startConsumers()
	if err != nil {
		Log.Error.Print("Kafka client error - ", err)
		return
	}

	defer func() {
		cons.client.Close()
		cons.dumpIndex()
	}()

	cons.TickerControlLoop(cons.persistTimeout, nil, cons.dumpIndex)
}
