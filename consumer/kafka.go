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
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	kafka "github.com/Shopify/sarama"
	cluster "github.com/bsm/sarama-cluster"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tsync"
)

const (
	kafkaOffsetNewest = "newest"
	kafkaOffsetOldest = "oldest"
)

// Kafka consumer plugin
//
// Thes consumer reads data from a given kafka topic. It is based on the sarama
// library so most settings are mapped to the settings from this library.
//
// Configuration example
//
//  consumerKafka:
//  	type: consumer.Kafka
//    	Topic: "default"
//    	ClientId: "gollum"
//    	Version: "0.8.2"
//    	GroupId: ""
//    	DefaultOffset: "newest"
//    	OffsetFile: ""
//    	FolderPermissions: "0755"
//    	Ordered: true
//    	MaxOpenRequests: 5
//    	ServerTimeoutSec: 30
//    	MaxFetchSizeByte: 0
//    	MinFetchSizeByte: 1
//    	FetchTimeoutMs: 250
//    	MessageBufferCount: 256
//    	PresistTimoutMs: 5000
//    	ElectRetries: 3
//    	ElectTimeoutMs: 250
//    	MetadataRefreshMs: 10000
//    	TlsEnabled: true
//    	TlsKeyLocation: ""
//    	TlsCertificateLocation: ""
//    	TlsCaLocation: ""
//    	TlsServerName: ""
//    	TlsInsecureSkipVerify: false
//    	SaslEnabled: false
//    	SaslUsername: "gollum"
//    	SaslPassword: ""
//    	Servers:
//        - "localhost:9092"
//
// Topic defines the kafka topic to read from. By default this is set to "default".
//
// ClientId sets the client id of this consumer. By default this is "gollum".
//
// GroupId sets the consumer group of this consumer. By default this is "" which
// disables consumer groups. This requires Version to be >= 0.9.
//
// Version defines the kafka protocol version to use. Common values are 0.8.2,
// 0.9.0 or 0.10.0. Values of the form "A.B" are allowed as well as "A.B.C"
// and "A.B.C.D". Defaults to "0.8.2", or if GroupId is set "0.9.0.1". If the
// version given is not known, the closest possible version is chosen. If GroupId
// is set and this is < "0.9", "0.9.0.1" will be used.
//
// DefaultOffset defines where to start reading the topic. Valid values are
// "oldest" and "newest". If OffsetFile is defined the DefaultOffset setting
// will be ignored unless the file does not exist.
// By default this is set to "newest". Ignored when using GroupId.
//
// OffsetFile defines the path to a file that stores the current offset inside
// a given partition. If the consumer is restarted that offset is used to continue
// reading. By default this is set to "" which disables the offset file. Ignored
// when using GroupId.
//
// FolderPermissions is used to create the offset file path if necessary.
// Set to 0755 by default. Ignored when using GroupId.
//
// Ordered can be set to enforce partitions to be read one-by-one in a round robin
// fashion instead of reading in parallel from all partitions.
// Set to false by default. Ignored when using GroupId.
//
// MaxOpenRequests defines the number of simultaneous connections are allowed.
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
// By default this is set to 8192.
//
// PresistTimoutMs defines the time in milliseconds between writes to OffsetFile.
// By default this is set to 5000. Shorter durations reduce the amount of
// duplicate messages after a fail but increases I/O. When using GroupId this
// only controls how long to pause after receiving errors.
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
// TlsEnable defines whether to use TLS to communicate with brokers. Defaults
// to false.
//
// TlsKeyLocation defines the path to the client's private key (PEM) for used
// for authentication. Defaults to "".
//
// TlsCertificateLocation defines the path to the client's public key (PEM) used
// for authentication. Defaults to "".
//
// TlsCaLocation defines the path to CA certificate(s) for verifying the broker's
// key. Defaults to "".
//
// TlsServerName is used to verify the hostname on the server's certificate
// unless TlsInsecureSkipVerify is true. Defaults to "".
//
// TlsInsecureSkipVerify controls whether to verify the server's certificate
// chain and host name. Defaults to false.
//
// SaslEnable is whether to use SASL for authentication. Defaults to false.
//
// SaslUsername is the user for SASL/PLAIN authentication. Defaults to "gollum".
//
// SaslPassword is the password for SASL/PLAIN authentication. Defaults to "".
//
// Servers contains the list of all kafka servers to connect to. By default this
// is set to contain only "localhost:9092".
type Kafka struct {
	core.SimpleConsumer `gollumdoc:"embed_type"`
	servers             []string      `config:"Servers" default:"localhost:9092"`
	topic               string        `config:"Topic" default:"default"`
	group               string        `config:"GroupId"`
	offsetFile          string        `config:"OffsetFile"`
	persistTimeout      time.Duration `config:"PresistTimoutMs" default:"5000" metric:"ms"`
	orderedRead         bool          `config:"Ordered"`
	folderPermissions   os.FileMode   `config:"FolderPermissions" default:"0755"`
	client              kafka.Client
	config              *kafka.Config
	groupClient         *cluster.Client
	groupConfig         *cluster.Config
	consumer            kafka.Consumer
	defaultOffset       int64
	offsets             map[int32]*int64
	MaxPartitionID      int32
}

func init() {
	core.TypeRegistry.Register(Kafka{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *Kafka) Configure(conf core.PluginConfigReader) {
	cons.offsets = make(map[int32]*int64)
	cons.MaxPartitionID = 0

	cons.config = kafka.NewConfig()
	cons.config.ClientID = conf.GetString("ClientId", "gollum")
	cons.config.ChannelBufferSize = int(conf.GetInt("MessageBufferCount", 8192))

	switch ver := conf.GetString("Version", "0.8.2"); ver {
	case "0.8.2.0":
		cons.config.Version = kafka.V0_8_2_0
	case "0.8.2.1":
		cons.config.Version = kafka.V0_8_2_1
	case "0.8", "0.8.2", "0.8.2.2":
		cons.config.Version = kafka.V0_8_2_2
	case "0.9.0", "0.9.0.0":
		cons.config.Version = kafka.V0_9_0_0
	case "0.9", "0.9.0.1":
		cons.config.Version = kafka.V0_9_0_1
	case "0.10", "0.10.0", "0.10.0.0":
		cons.config.Version = kafka.V0_10_0_0
	default:
		cons.Logger.Warning("Unknown kafka version given: ", ver)
		parts := strings.Split(ver, ".")
		if len(parts) < 2 {
			cons.config.Version = kafka.V0_8_2_2
		} else {
			minor, _ := strconv.ParseUint(parts[1], 10, 8)
			switch {
			case minor <= 8:
				cons.config.Version = kafka.V0_8_2_2
			case minor == 9:
				cons.config.Version = kafka.V0_9_0_1
			case minor >= 10:
				cons.config.Version = kafka.V0_10_0_0
			}
		}
	}

	cons.config.Net.MaxOpenRequests = int(conf.GetInt("MaxOpenRequests", 5))
	cons.config.Net.DialTimeout = time.Duration(conf.GetInt("ServerTimeoutSec", 30)) * time.Second
	cons.config.Net.ReadTimeout = cons.config.Net.DialTimeout
	cons.config.Net.WriteTimeout = cons.config.Net.DialTimeout

	cons.config.Net.TLS.Enable = conf.GetBool("TlsEnable", false)
	if cons.config.Net.TLS.Enable {
		cons.config.Net.TLS.Config = &tls.Config{}

		keyFile := conf.GetString("TlsKeyLocation", "")
		certFile := conf.GetString("TlsCertificateLocation", "")
		if keyFile != "" && certFile != "" {
			cert, err := tls.LoadX509KeyPair(certFile, keyFile)
			if conf.Errors.Push(err) {
				return
			}
			cons.config.Net.TLS.Config.Certificates = []tls.Certificate{cert}
		} else if certFile == "" {
			conf.Errors.Pushf("Cannot specify TlsKeyLocation without TlsCertificateLocation")
			return
		} else if keyFile == "" {
			conf.Errors.Pushf("Cannot specify TlsCertificateLocation without TlsKeyLocation")
			return
		}

		caFile := conf.GetString("TlsCaLocation", "")
		if caFile == "" {
			conf.Errors.Pushf("TlsEnable is set to true, but no TlsCaLocation was specified")
			return
		}

		caCert, err := ioutil.ReadFile(caFile)
		if conf.Errors.Push(err) {
			return
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		cons.config.Net.TLS.Config.RootCAs = caCertPool

		serverName := conf.GetString("TlsServerName", "")
		if serverName != "" {
			cons.config.Net.TLS.Config.ServerName = serverName
		}

		cons.config.Net.TLS.Config.InsecureSkipVerify = conf.GetBool("TlsInsecureSkipVerify", false)
	}

	cons.config.Net.SASL.Enable = conf.GetBool("SaslEnable", false)
	if cons.config.Net.SASL.Enable {
		cons.config.Net.SASL.User = conf.GetString("SaslUser", "gollum")
		cons.config.Net.SASL.Password = conf.GetString("SaslPassword", "")
	}

	cons.config.Metadata.Retry.Max = int(conf.GetInt("ElectRetries", 3))
	cons.config.Metadata.Retry.Backoff = time.Duration(conf.GetInt("ElectTimeoutMs", 250)) * time.Millisecond
	cons.config.Metadata.RefreshFrequency = time.Duration(conf.GetInt("MetadataRefreshMs", 10000)) * time.Millisecond

	cons.config.Consumer.Fetch.Min = int32(conf.GetInt("MinFetchSizeByte", 1))
	cons.config.Consumer.Fetch.Max = int32(conf.GetInt("MaxFetchSizeByte", 0))
	cons.config.Consumer.Fetch.Default = cons.config.Consumer.Fetch.Max
	cons.config.Consumer.MaxWaitTime = time.Duration(conf.GetInt("FetchTimeoutMs", 250)) * time.Millisecond

	if cons.group != "" {
		cons.offsetFile = "" // forcibly ignore this option
		switch cons.config.Version {
		case kafka.V0_8_2_0, kafka.V0_8_2_1, kafka.V0_8_2_2:
			cons.Logger.Warning("Invalid kafka version 0.8.x given, minimum is 0.9 for consumer groups, defaulting to 0.9.0.1")
			cons.config.Version = kafka.V0_9_0_1
		}

		cons.groupConfig = cluster.NewConfig()
		cons.groupConfig.Config = *cons.config
	}

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
			cons.Logger.Errorf("Failed to open kafka offset file: %s", err.Error())
		} else {
			// Decode the JSON file into the partition -> offset map
			encodedOffsets := make(map[string]int64)
			err = json.Unmarshal(fileContents, &encodedOffsets)
			if conf.Errors.Push(err) {
				return
			}

			for k, v := range encodedOffsets {
				id, err := strconv.Atoi(k)
				if conf.Errors.Push(err) {
					return
				}
				startOffset := v
				cons.offsets[int32(id)] = &startOffset
			}
		}
	}

	kafka.Logger = cons.Logger // TODO: kafka.logger = cons.Log.Note => ?
}

func (cons *Kafka) restartGroup() {
	time.Sleep(cons.persistTimeout)
	cons.readFromGroup()
}

// Main fetch loop for kafka events
func (cons *Kafka) readFromGroup() {
	consumer, err := cluster.NewConsumerFromClient(cons.groupClient, cons.group, []string{cons.topic})
	if err != nil {
		defer cons.restartGroup()
		cons.Logger.Errorf("Restarting kafka consumer (%s:%s) - %s", cons.topic, cons.group, err.Error())
		return // ### return, stop and retry ###
	}

	// Make sure we wait for all consumers to end
	cons.AddWorker()
	defer func() {
		if !cons.groupClient.Closed() {
			consumer.Close()
		}
		cons.WorkerDone()
	}()

	// Loop over worker
	spin := tsync.NewSpinner(tsync.SpinPriorityLow)

	for !cons.groupClient.Closed() {
		select {
		case event := <-consumer.Messages():
			cons.enqueueEvent(event)

		case err := <-consumer.Errors():
			defer cons.restartGroup()
			cons.Logger.Error("Kafka consumer error:", err)
			return // ### return, try reconnect ###

		default:
			spin.Yield()
		}
	}
}

func (cons *Kafka) startConsumerForPartition(partitionID int32) kafka.PartitionConsumer {
	for !cons.client.Closed() {
		startOffset := atomic.LoadInt64(cons.offsets[partitionID])
		consumer, err := cons.consumer.ConsumePartition(cons.topic, partitionID, startOffset)
		if err == nil {
			return consumer // ### return, success ###
		}

		cons.Logger.Errorf("Failed to start kafka consumer (%s:%d) - %s", cons.topic, startOffset, err.Error())

		// Reset offset to default value if we have an offset error
		if err == kafka.ErrOffsetOutOfRange {
			// Actually we would need to see if we're out of range at the end or at the start
			// and choose OffsetOldest or OffsetNewset accordingly.
			// At the moment we stick to the most common case here.
			startOffset = kafka.OffsetOldest
			atomic.StoreInt64(cons.offsets[partitionID], startOffset)
		} else {
			time.Sleep(cons.persistTimeout)
		}
	}

	return nil
}

// Main fetch loop for kafka events
func (cons *Kafka) readFromPartition(partitionID int32) {
	cons.AddWorker()
	defer cons.WorkerDone()

	partCons := cons.startConsumerForPartition(partitionID)
	spin := tsync.NewSpinner(tsync.SpinPriorityLow)

	for !cons.client.Closed() {

		select {
		case event := <-partCons.Messages():
			//Added some verbose information so that we can investigate reasons of
			//exception. Probably it might happen when sarama close the channel
			//so we will get nil message from the channel.
			if event == nil || cons.offsets == nil || cons.offsets[partitionID] == nil {
				cons.Logger.Errorf("Kafka consumer failed to store offset. Trace : event : %+v, cons.partCons: %+v, partitionID: %d\n",
					event, cons.offsets, partitionID)

				partCons.Close()
				partCons = cons.startConsumerForPartition(partitionID)
				continue
			}

			atomic.StoreInt64(cons.offsets[partitionID], event.Offset)
			cons.enqueueEvent(event)

		case err := <-partCons.Errors():
			cons.Logger.Error("Kafka consumer error:", err)
			if !cons.client.Closed() {
				partCons.Close()
			}
			partCons = cons.startConsumerForPartition(partitionID)

		default:
			spin.Yield()
		}
	}
}

func (cons *Kafka) readPartitions(partitions []int32) {
	cons.AddWorker()
	defer cons.WorkerDone()

	// Start consumers

	consumers := []kafka.PartitionConsumer{}
	for _, partitionID := range partitions {
		consumer := cons.startConsumerForPartition(partitionID)
		consumers = append(consumers, consumer)
	}

	// Loop over worker.
	// Note: partitions and consumer are assumed to be index parallel

	spin := tsync.NewSpinner(tsync.SpinPriorityLow)
	for !cons.client.Closed() {
		for idx, consumer := range consumers {
			partition := partitions[idx]

			select {
			case event := <-consumer.Messages():
				atomic.StoreInt64(cons.offsets[partition], event.Offset)
				cons.enqueueEvent(event)

			case err := <-consumer.Errors():
				cons.Logger.Error("Kafka consumer error:", err)
				if !cons.client.Closed() {
					consumer.Close()
				}

				consumer = cons.startConsumerForPartition(partition)
				consumers[idx] = consumer

			default:
				spin.Yield()
			}
		}
	}
}

func (cons *Kafka) enqueueEvent(event *kafka.ConsumerMessage) {
	metaData := core.Metadata{}

	metaData.SetValue("topic", []byte(event.Topic))
	metaData.SetValue("key", event.Key)

	cons.EnqueueWithMetadata(event.Value, metaData)
}

func (cons *Kafka) startReadTopic(topic string) {
	partitions, err := cons.client.Partitions(topic)
	if err != nil {
		cons.Logger.Error(err)
		time.AfterFunc(cons.persistTimeout, func() { cons.startReadTopic(topic) })
		return
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
}

// Start one consumer per partition as a go routine
func (cons *Kafka) startAllConsumers() error {
	var err error

	if cons.group != "" {
		cons.groupClient, err = cluster.NewClient(cons.servers, cons.groupConfig)
		if err != nil {
			return err
		}

		go cons.readFromGroup()
		return nil // ### return, group processing ###
	}

	cons.client, err = kafka.NewClient(cons.servers, cons.config)
	if err != nil {
		return err
	}

	cons.consumer, err = kafka.NewConsumerFromClient(cons.client)
	if err != nil {
		return err
	}

	cons.startReadTopic(cons.topic)

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
			cons.Logger.Error("Kafka index file write error - ", err)
		} else {
			fileDir := path.Dir(cons.offsetFile)
			if err := os.MkdirAll(fileDir, 0755); err != nil {
				cons.Logger.Errorf("Failed to create %s because of %s", fileDir, err.Error())
			} else {
				ioutil.WriteFile(cons.offsetFile, data, 0644)
			}
		}
	}
}

// Consume starts a kafka consumer per partition for this topic
func (cons *Kafka) Consume(workers *sync.WaitGroup) {
	cons.SetWorkerWaitGroup(workers)

	if err := cons.startAllConsumers(); err != nil {
		cons.Logger.Error("Kafka client error - ", err)
		time.AfterFunc(cons.config.Net.DialTimeout, func() { cons.Consume(workers) })
		return
	}

	defer func() {
		cons.client.Close()
		cons.dumpIndex()
	}()

	cons.TickerControlLoop(cons.persistTimeout, cons.dumpIndex)
}
