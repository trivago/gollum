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

// Kafka consumer
//
// This consumer reads data from a kafka topic. It is based on the sarama
// library; most settings are mapped to the settings from this library.
//
// Metadata
//
// *NOTE: The metadata will only set if the parameter `SetMetadata` is active.*
//
// - topic: Contains the name of the kafka topic
//
// - key: Contains the key of the kafka message
//
// Parameters
//
// - Servers: Defines the list of all kafka brokers to initially connect to when
// querying topic metadata. This list requires at least one borker to work and
// ideally contains all the brokers in the cluster.
// By default this parameter is set to ["localhost:9092"].
//
// - Topic: Defines the kafka topic to read from.
// By default this parameter is set to "default".
//
// - ClientId: Sets the client id used in requests by this consumer.
// By default this parameter is set to "gollum".
//
// - GroupId: Sets the consumer group of this consumer. If empty, consumer
// groups are not used. This setting requires Kafka version >= 0.9.
// By default this parameter is set to "".
//
// - Version: Defines the kafka protocol version to use. Common values are 0.8.2,
// 0.9.0 or 0.10.0. Values of the form "A.B" are allowed as well as "A.B.C"
// and "A.B.C.D". If the version given is not known, the closest possible
// version is chosen. If GroupId is set to a value < "0.9", "0.9.0.1" will be used.
// By default this parameter is set to "0.8.2".
//
// - SetMetadata: When this value is set to "true", the fields mentioned in the metadata
// section will be added to each message. Adding metadata will have a
// performance impact on systems with high throughput.
// By default this parameter is set to "false".
//
// - DefaultOffset: Defines the initial offest when starting to read the topic.
// Valid values are "oldest" and "newest". If OffsetFile
// is defined and the file exists, the DefaultOffset parameter is ignored.
// If GroupId is defined, this setting will only be used for the first request.
// By default this parameter is set to "newest".
//
// - OffsetFile: Defines the path to a file that holds the current offset of a
// given partition. If the consumer is restarted, reading continues from that
// offset. To disable this setting, set it to "". Please note that offsets
// stored in the file might be outdated. In that case DefaultOffset "oldest"
// will be used.
// By default this parameter is set to "".
//
// - FolderPermissions: Used to create the path to the offset file if necessary.
// By default this parameter is set to "0755".
//
// - Ordered: Forces partitions to be read one-by-one in a round robin fashion
// instead of reading them all in parallel. Please note that this may restore
// the original ordering but does not necessarily do so. The term "ordered" refers
// to an ordered reading of all partitions, as opposed to reading them randomly.
// By default this parameter is set to false.
//
// - MaxOpenRequests: Defines the number of simultaneous connections to a
// broker at a time.
// By default this parameter is set to 5.
//
// - ServerTimeoutSec: Defines the time after which a connection will time out.
// By default this parameter is set to 30.
//
// - MaxFetchSizeByte: Sets the maximum size of a message to fetch. Larger
// messages will be ignored. When set to 0 size of the messages is ignored.
// By default this parameter is set to 0.
//
// - MinFetchSizeByte: Defines the minimum amout of data to fetch from Kafka per
// request. If less data is available the broker will wait.
// By default this parameter is set to 1.
//
// - DefaultFetchSizeByte: Defines the average amout of data to fetch per
// request. This value must be greater than 0.
// By default this parameter is set to 32768.
//
// - FetchTimeoutMs: Defines the time in milliseconds to wait on reaching
// MinFetchSizeByte before fetching new data regardless of size.
// By default this parameter is set to 250.
//
// - MessageBufferCount: Sets the internal channel size for the kafka client.
// By default this parameter is set to 8192.
//
// - PresistTimoutMs: Defines the interval in milliseconds in which data is
// written to the OffsetFile. A short duration reduces the amount of duplicate
// messages after a crash but increases I/O. When using GroupId this setting
// controls the pause time after receiving errors.
// By default this parameter is set to 5000.
//
// - ElectRetries: Defines how many times to retry fetching the new master
// partition during a leader election.
// By default this parameter is set to 3.
//
// - ElectTimeoutMs: Defines the number of milliseconds to wait for the cluster
// to elect a new leader.
// By default this parameter is set to 250.
//
// - MetadataRefreshMs: Defines the interval in milliseconds used for fetching
// kafka metadata from the cluster (e.g. number of partitons).
// By default this parameter is set to 10000.
//
// - TlsEnable: Defines whether to use TLS based authentication when
// communicating with brokers.
// By default this parameter is set to false.
//
// - TlsKeyLocation: Defines the path to the client's PEM-formatted private key
// used for TLS based authentication.
// By default this parameter is set to "".
//
// - TlsCertificateLocation: Defines the path to the client's PEM-formatted
// public key used for TLS based authentication.
// By default this parameter is set to "".
//
// - TlsCaLocation: Defines the path to the CA certificate(s) for verifying a
// broker's key when using TLS based authentication.
// By default this parameter is set to "".
//
// - TlsServerName: Defines the expected hostname used by hostname verification
// when using TlsInsecureSkipVerify.
// By default this parameter is set to "".
//
// - TlsInsecureSkipVerify: Enables verification of the server's certificate
// chain and host name.
// By default this parameter is set to false.
//
// - SaslEnable:Defines whether to use SASL based authentication when
// communicating with brokers.
// By default this parameter is set to false.
//
// - SaslUsername: Defines the username for SASL/PLAIN authentication.
// By default this parameter is set to "gollum".
//
// - SaslPassword: Defines the password for SASL/PLAIN authentication.
// By default this parameter is set to "".
//
// Examples
//
// This config reads the topic "logs" from a cluster with 4 brokers.
//
//  kafkaIn:
//    Type: consumer.Kafka
//    Streams: logs
//    Topic: logs
//    ClientId: "gollum log reader"
//    DefaultOffset: newest
//    OffsetFile: /var/gollum/logs.offset
//    Servers:
//      - "kafka0:9092"
//      - "kafka1:9092"
//      - "kafka2:9092"
//      - "kafka3:9092"
type Kafka struct {
	core.SimpleConsumer `gollumdoc:"embed_type"`
	client              kafka.Client
	consumer            kafka.Consumer
	config              *kafka.Config
	groupClient         *cluster.Client
	groupConfig         *cluster.Config
	offsets             map[int32]*int64
	servers             []string `config:"Servers"`
	topic               string   `config:"Topic" default:"default"`
	group               string   `config:"GroupId"`
	offsetFile          string   `config:"OffsetFile"`
	defaultOffset       int64
	persistTimeout      time.Duration `config:"PresistTimoutMs" default:"5000" metric:"ms"`
	folderPermissions   os.FileMode   `config:"FolderPermissions" default:"0755"`
	MaxPartitionID      int32
	orderedRead         bool `config:"Ordered"`
	hasToSetMetadata    bool `config:"SetMetadata" default:"false"`
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
	case "0.10.0.1":
		cons.config.Version = kafka.V0_10_0_1
	case "0.10.1", "0.10.1.0":
		cons.config.Version = kafka.V0_10_1_0
	case "0.10.2", "0.10.2.0":
		cons.config.Version = kafka.V0_10_2_0
	case "0.11", "0.11.0", "0.11.0.0":
		cons.config.Version = kafka.V0_11_0_0
	case "1", "1.0", "1.0.0", "1.0.0.0":
		cons.config.Version = kafka.V1_0_0_0
	default:
		cons.Logger.Warningf("Unknown kafka version given: %s. Falling back to 0.8.2", ver)
		cons.config.Version = kafka.V0_8_2_2
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
	cons.config.Consumer.Fetch.Default = int32(conf.GetInt("DefaultFetchSizeByte", 32768))
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
			cons.Logger.Warningf("Failed to open kafka offset file: %s", err.Error())
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

	kafka.Logger = cons.Logger.WithField("Scope", "Sarama")
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
		case event, ok := <-consumer.Messages():
			if ok {
				cons.enqueueEvent(event)
				consumer.MarkOffset(event, "")
			}

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
	if cons.hasToSetMetadata {
		metaData := core.NewMetadata()

		metaData.Set("topic", event.Topic)
		metaData.Set("key", event.Key)

		cons.EnqueueWithMetadata(event.Value, metaData)
	} else {
		cons.SimpleConsumer.Enqueue(event.Value)
	}
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
			cons.Logger.WithError(err).Error("Kafka index file write error")
			return
		}

		fileDir := path.Dir(cons.offsetFile)
		if err := os.MkdirAll(fileDir, cons.folderPermissions); err != nil {
			cons.Logger.WithError(err).Errorf("Failed to create %s", fileDir)
			return
		}

		if err := ioutil.WriteFile(cons.offsetFile, data, 0644); err != nil {
			cons.Logger.WithError(err).Error("Failed to write offsets")
		}
	}
}

// Consume starts a kafka consumer per partition for this topic
func (cons *Kafka) Consume(workers *sync.WaitGroup) {
	cons.SetWorkerWaitGroup(workers)

	if err := cons.startAllConsumers(); err != nil {
		cons.Logger.WithError(err).Error("Kafka client error")
		time.AfterFunc(cons.config.Net.DialTimeout, func() { cons.Consume(workers) })
		return
	}

	defer func() {
		cons.client.Close()
		cons.dumpIndex()
	}()

	cons.TickerControlLoop(cons.persistTimeout, cons.dumpIndex)
}
