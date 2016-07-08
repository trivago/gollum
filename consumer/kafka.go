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

package consumer

import (
	"encoding/json"
	kafka "github.com/Shopify/sarama"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tsync"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	kafkaOffsetNewest = "newest"
	kafkaOffsetOldest = "oldest"
)

// Kafka consumer plugin
// Thes consumer reads data from a given kafka topic. It is based on the sarama
// library so most settings are mapped to the settings from this library.
// When attached to a fuse, this consumer will stop processing messages in case
// that fuse is burned.
// Configuration example
//
//  - "consumer.Kafka":
//    Topic: "default"
//    DefaultOffset: "newest"
//    OffsetFile: ""
//	  FolderPermissions: "0755"
//    Ordered: true
//    MaxOpenRequests: 5
//    ServerTimeoutSec: 30
//    MaxFetchSizeByte: 0
//    MinFetchSizeByte: 1
//    FetchTimeoutMs: 250
//    MessageBufferCount: 256
//    PresistTimoutMs: 5000
//    ElectRetries: 3
//    ElectTimeoutMs: 250
//    MetadataRefreshMs: 10000
//    PrependKey: false
//    KeySeparator: ":"
//    Servers:
//      - "localhost:9092"
//
// Topic defines the kafka topic to read from. By default this is set to "default".
//
// DefaultOffset defines where to start reading the topic. Valid values are
// "oldest" and "newest". If OffsetFile is defined the DefaultOffset setting
// will be ignored unless the file does not exist.
// By default this is set to "newest".
//
// OffsetFile defines the path to a file that stores the current offset inside
// a given partition. If the consumer is restarted that offset is used to continue
// reading. By default this is set to "" which disables the offset file.
//
// FolderPermissions is used to create the offset file path if necessary.
// Set to 0755 by default.
//
// Ordered can be set to enforce partitions to be read one-by-one in a round robin
// fashion instead of reading in parallel from all partitions.
// Set to false by default.
//
// PrependKey can be enabled to prefix the read message with the key from the
// kafka message. A separator will ba appended to the key. See KeySeparator.
// By default this is option set to false.
//
// KeySeparator defines the separator that is appended to the kafka message key
// if PrependKey is set to true. Set to ":" by default.
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
// Servers contains the list of all kafka servers to connect to. By default this
// is set to contain only "localhost:9092".
type Kafka struct {
	core.SimpleConsumer
	servers        []string
	topic          string
	client         kafka.Client
	config         *kafka.Config
	consumer       kafka.Consumer
	offsetFile     string
	defaultOffset  int64
	offsets        map[int32]*int64
	MaxPartitionID int32
	persistTimeout time.Duration
	orderedRead    bool
	prependKey     bool
	keySeparator   []byte
	folderPermissions os.FileMode
}

func init() {
	core.TypeRegistry.Register(Kafka{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *Kafka) Configure(conf core.PluginConfigReader) error {
	cons.SimpleConsumer.Configure(conf)

	cons.servers = conf.GetStringArray("Servers", []string{"localhost:9092"})
	cons.topic = conf.GetString("Topic", "default")
	cons.offsetFile = conf.GetString("OffsetFile", "")
	cons.persistTimeout = time.Duration(conf.GetInt("PresistTimoutMs", 5000)) * time.Millisecond
	cons.orderedRead = conf.GetBool("Ordered", false)
	cons.offsets = make(map[int32]*int64)
	cons.MaxPartitionID = 0
	cons.keySeparator = []byte(conf.GetString("KeySeparator", ":"))
	cons.prependKey = conf.GetBool("PrependKey", false)

	folderFlags, err := strconv.ParseInt(conf.GetString("FolderPermissions", "0755"), 8, 32)
	cons.folderPermissions = os.FileMode(folderFlags)
	if err != nil {
		return err
	}

	cons.config = kafka.NewConfig()
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
	cons.config.Consumer.Fetch.Default = cons.config.Consumer.Fetch.Max
	cons.config.Consumer.MaxWaitTime = time.Duration(conf.GetInt("FetchTimeoutMs", 250)) * time.Millisecond

	offsetValue := strings.ToLower(conf.GetString("DefaultOffset", kafkaOffsetNewest))
	switch offsetValue {
	case kafkaOffsetNewest:
		cons.defaultOffset = kafka.OffsetNewest

	case kafkaOffsetOldest:
		cons.defaultOffset = kafka.OffsetOldest

	default:
		cons.defaultOffset, _ = strconv.ParseInt(offsetValue, 10, 64)
	}

	if cons.offsetFile != "" {
		fileContents, err := ioutil.ReadFile(cons.offsetFile)
		if err != nil {
			Log.Error.Printf("Failed to open kafka offset file: %s", err.Error())
		} else {
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
			startOffset := v
			cons.offsets[int32(id)] = &startOffset
			}
		}
	}

	kafka.Logger = cons.Log.Note
	return conf.Errors.OrNil()
}

func (cons *Kafka) restartPartition(partitionID int32) {
	time.Sleep(cons.persistTimeout)
	cons.readFromPartition(partitionID)
}

func (cons *Kafka) keyedMessage(key []byte, value []byte) []byte {
	buffer := make([]byte, len(key)+len(cons.keySeparator)+len(value))
	offset := copy(buffer, key)
	offset += copy(buffer[offset:], cons.keySeparator)
	copy(buffer[offset:], value)
	return buffer
}

// Main fetch loop for kafka events
func (cons *Kafka) readFromPartition(partitionID int32) {
	currentOffset := atomic.LoadInt64(cons.offsets[partitionID])
	partCons, err := cons.consumer.ConsumePartition(cons.topic, partitionID, currentOffset)
	if err != nil {
		defer cons.restartPartition(partitionID)
		cons.Log.Error.Printf("Restarting kafka consumer (%s:%d) - %s", cons.topic, currentOffset, err.Error())
		return // ### return, stop and retry ###
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
	spin := tsync.NewSpinner(tsync.SpinPriorityLow)

	for !cons.client.Closed() {
		cons.WaitOnFuse()
		select {
		case event := <-partCons.Messages():
			atomic.StoreInt64(cons.offsets[partitionID], event.Offset)
			if cons.prependKey {
				cons.Enqueue(cons.keyedMessage(event.Key, event.Value))
			} else {
				cons.Enqueue(event.Value)
			}

		case err := <-partCons.Errors():
			defer cons.restartPartition(partitionID)
			cons.Log.Error.Print("Kafka consumer error:", err)
			return // ### return, try reconnect ###

		default:
			spin.Yield()
		}
	}
}

func (cons *Kafka) readPartitions(partitions []int32) {
	consumers := []kafka.PartitionConsumer{}

	// Start consumer
initLoop:
	for _, partitionID := range partitions {
		startOffset := atomic.LoadInt64(cons.offsets[partitionID])
		consumer, err := cons.consumer.ConsumePartition(cons.topic, partitionID, startOffset)

		// Retry consumer until successful
		for err != nil {
			cons.Log.Error.Printf("Failed to start kafka consumer (%s:%d) - %s", cons.topic, partitionID, err.Error())
			time.Sleep(cons.persistTimeout)
			consumer, err = cons.consumer.ConsumePartition(cons.topic, partitionID, startOffset)
			if cons.client.Closed() {
				break initLoop
			}
		}

		consumers = append(consumers, consumer)
	}

	// Make sure we wait for all consumers to end
	cons.AddWorker()
	defer func() {
		if !cons.client.Closed() {
			for _, consumer := range consumers {
				consumer.Close()
			}
		}
		cons.WorkerDone()
	}()

	// Loop over worker.
	// Note: partitions and consumer are assumed to be index parallel
	for !cons.client.Closed() {
		for idx, consumer := range consumers {
			var err error
			cons.WaitOnFuse()
			partition := partitions[idx]

			select {
			case event := <-consumer.Messages():
				atomic.StoreInt64(cons.offsets[partition], event.Offset)
				if cons.prependKey {
					cons.Enqueue(cons.keyedMessage(event.Key, event.Value))
				} else {
					cons.Enqueue(event.Value)
				}

			case err = <-consumer.Errors():
				cons.Log.Error.Print("Kafka consumer error:", err)
				consumer.Close()

			reconnect:
				consumer, err = cons.consumer.ConsumePartition(cons.topic, partition, atomic.LoadInt64(cons.offsets[partition]))
				if err != nil {
					cons.Log.Error.Printf("Failed to restart kafka consumer (%s:%d) - %s", cons.topic, partition, err.Error())
					time.Sleep(cons.persistTimeout)
					goto reconnect
				}
			}
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

	for _, partitionID := range partitions {
		if _, mapped := cons.offsets[partitionID]; !mapped {
			startOffset := cons.defaultOffset
			cons.offsets[partitionID] = &startOffset
		}
		if partitionID > cons.MaxPartitionID {
			cons.MaxPartitionID = partitionID
		}
	}

	if cons.orderedRead {
		go cons.readPartitions(partitions)
	} else {
		for _, partitionID := range partitions {
			go cons.readFromPartition(partitionID)
		}
	}

	return nil
}

// Write index file to disc
func (cons *Kafka) dumpIndex() {
	if cons.offsetFile != "" {
		encodedOffsets := make(map[string]int64)
		for k := range cons.offsets {
			encodedOffsets[strconv.Itoa(int(k))] = atomic.LoadInt64(cons.offsets[k])
		}

		data, err := json.Marshal(encodedOffsets)
		if err != nil {
			cons.Log.Error.Print("Kafka index file write error - ", err)
		} else {
			fileDir := path.Dir(cons.offsetFile)
			if err := os.MkdirAll(fileDir, 0755); err != nil {
				Log.Error.Printf("Failed to create %s because of %s", fileDir, err.Error())
			} else {
				ioutil.WriteFile(cons.offsetFile, data, 0644)
			}
		}
	}
}

// Consume starts a kafka consumer per partition for this topic
func (cons *Kafka) Consume(workers *sync.WaitGroup) {
	cons.SetWorkerWaitGroup(workers)

	if err := cons.startConsumers(); err != nil {
		cons.Log.Error.Print("Kafka client error - ", err)
		return
	}

	defer func() {
		cons.client.Close()
		cons.dumpIndex()
	}()

	cons.TickerControlLoop(cons.persistTimeout, cons.dumpIndex)
}
