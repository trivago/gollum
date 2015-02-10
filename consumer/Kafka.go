package consumer

import (
	"fmt"
	kafka "github.com/Shopify/sarama"
	"github.com/trivago/gollum/log"
	"github.com/trivago/gollum/shared"
	"io/ioutil"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	offsetNewset = "Newest"
	offsetOldest = "Oldest"
	offsetFile   = "File"
)

// Kafka consumer plugin
// Configuration example
//
// - "consumer.Kafka":
//   Enable: true
//   ClientID: "logger"
//   ConsumerGroup: "logreader"
//   MaxFetchSizeByte: 8192
//   MinFetchSizeByte: 0
//   MaxMessageSizeByte: 0
//   FetchTimeoutMs: 500
//   MessageBufferCount: 32
//   ElectTimeoutMs: 1000
//   MetadataRefreshSec: 30
//   Offset: "File"
//   OffsetFile: "/tmp/gollum_kafka.idx"
//
// ClientId sets the client id of this producer. By default this is "gollum".
//
// ConsumerGroup sets the consumer group of this consumer. By default this is
// set to "gollum".
//
// MaxFetchSizeByte the maximum amount of bytes to fetch from Kafka per request.
// By default this is set to 32768.
//
// MinFetchSizeByte defines the minimum amout of data to fetch from Kafka per
// request. If less data is available the broker will wait. By default this is
// set to 1.
//
// MaxMessageSizeByte sets the maximum size of a message to fetch. Larger messages
// will be ignored. By default this is set to 0 (fetch all messages).
//
// FetchTimeoutMs defines the time in milliseconds the broker will wait for
// MinFetchSizeByte to be reached before processing data anyway. By default this
// is set to 250ms.
//
// MessageBufferCount defines the number of events to load in the background
// while the consumer is processing messages. By default this is set to 16,
// setting it to 0 disables background fetching
//
// Offset defines the message index to start reading from. Valid values are either
// (case sensitive) "Newset", "Oldest", or "File". If "File" is used the
// OffsetFile option must be set. The default value is "Newest"
//
// OffsetFile defines a path to a file containing the current index per topic
// partition. This file is used by Offset: "file" and is disabled by default.
// If no file exists all partitions start at "Newest"
//
// OffsetTimeoutMs defines the time in milliseconds between writes to OffsetFile.
// By default this is set to 5000. Shorter durations reduce the amount of
// duplicate messages after a fail but increases I/O.
//
// ElectTimeoutMs defines the number of milliseconds to wait for the cluster to
// elect a new leader. Defaults to 250.
//
// MetadataRefreshSec set the interval in seconds for fetching cluster metadata.
// By default this is set to 10.
type Kafka struct {
	standardConsumer
	servers        []string
	topic          string
	clientID       string
	consumerGroup  string
	client         *kafka.Client
	clientConfig   *kafka.ClientConfig
	consumer       *kafka.Consumer
	consumerConfig *kafka.ConsumerConfig
	offsetFile     string
	offsets        []int64
	offsetTimeout  time.Duration
	quit           bool
}

func init() {
	shared.RuntimeType.Register(Kafka{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *Kafka) Configure(conf shared.PluginConfig) error {
	err := cons.standardConsumer.Configure(conf)
	if err != nil {
		return err
	}

	cons.clientConfig = kafka.NewClientConfig()
	cons.consumerConfig = kafka.NewConsumerConfig()
	cons.servers = conf.GetStringArray("Servers", []string{})
	cons.topic = conf.GetString("Topic", "default")
	cons.clientID = conf.GetString("ClientId", "gollum")
	cons.consumerGroup = conf.GetString("ConsumerGroup", "gollum")
	cons.offsetFile = conf.GetString("OffsetFile", "")
	cons.offsetTimeout = time.Duration(conf.GetInt("OffsetTimoutMs", 5000)) * time.Millisecond

	cons.consumerConfig.DefaultFetchSize = int32(conf.GetInt("MaxFetchSizeByte", 32768))
	cons.consumerConfig.MinFetchSize = int32(conf.GetInt("MinFetchSizeByte", 1))
	cons.consumerConfig.MaxMessageSize = int32(conf.GetInt("MaxMessageSizeByte", 0))
	cons.consumerConfig.MaxWaitTime = time.Duration(conf.GetInt("FetchTimeoutMs", 250)) * time.Millisecond
	cons.consumerConfig.EventBufferSize = conf.GetInt("MessageBufferCount", 16)
	cons.consumerConfig.OffsetValue = 0

	switch conf.GetString("Offset", offsetNewset) {
	default:
		fallthrough
	case offsetNewset:
		cons.consumerConfig.OffsetMethod = kafka.OffsetMethodNewest

	case offsetOldest:
		cons.consumerConfig.OffsetMethod = kafka.OffsetMethodOldest

	case offsetFile:
		cons.consumerConfig.OffsetMethod = kafka.OffsetMethodManual
		fileContents, err := ioutil.ReadFile(cons.offsetFile)
		if err != nil {
			Log.Warning.Print(err)
		} else {
			offsets := strings.Split(string(fileContents), ",")
			cons.offsets = make([]int64, len(offsets))

			for idx, offset := range offsets {
				cons.offsets[idx], _ = strconv.ParseInt(offset, 10, 32)
			}
		}
	}

	cons.clientConfig.WaitForElection = time.Duration(conf.GetInt("ElectTimeoutMs", 250)) * time.Millisecond
	cons.clientConfig.BackgroundRefreshFrequency = time.Duration(conf.GetInt("MetadataRefreshSec", 10)) * time.Second

	return nil
}

// Restart the consumer after wating for offsetTimeout
func (cons *Kafka) restart(err error, offsetIdx int, partition int32) {
	Log.Error.Print("Kafka consumer error:", err)
	time.Sleep(cons.offsetTimeout)
	cons.fetch(offsetIdx, partition)
}

// Main fetch loop for kafka events
func (cons *Kafka) fetch(offsetIdx int, partition int32) {
	config := *cons.consumerConfig

	if len(cons.offsets) > offsetIdx {
		config.OffsetValue = cons.offsets[offsetIdx]
	} else {
		if config.OffsetMethod == kafka.OffsetMethodManual {
			config.OffsetMethod = kafka.OffsetMethodNewest
		}
	}

	consumer, err := kafka.NewConsumer(cons.client, cons.topic, partition, cons.consumerGroup, &config)
	if err != nil {
		if !cons.quit {
			go cons.restart(err, offsetIdx, partition)
		}
		return // ### return, stop this consumer ###
	}

	// Make sure we wait for all consumers to end

	cons.state.WaitGroup.Add(1)

	defer func() {
		consumer.Close()
		cons.state.WaitGroup.Done()
	}()

	for {
		event := <-consumer.Events()
		if event.Err != nil {
			if !cons.quit {
				go cons.restart(err, offsetIdx, partition)
			}
			return // ### return, stop this consumer ###
		}

		cons.offsets[offsetIdx] = int64(math.Max(float64(cons.offsets[offsetIdx]), float64(event.Offset)))
		cons.postMessageFromSlice(event.Value)
	}
}

// Start one consumer per partition as a go routine
func (cons *Kafka) startConsumers() error {
	var err error

	cons.client, err = kafka.NewClient(cons.clientID, cons.servers, cons.clientConfig)
	if err != nil {
		return err
	}

	partitions, err := cons.client.Partitions(cons.topic)
	if err != nil {
		return err
	}

	for idx, partition := range partitions {
		go cons.fetch(idx, partition)
	}

	return nil
}

// Write index file to disc
func (cons *Kafka) dumpIndex() {
	csvString := ""
	for _, value := range cons.offsets {
		csvString += fmt.Sprintf("%d,", value)
	}

	ioutil.WriteFile(cons.offsetFile, []byte(csvString), 0644)
}

// Consume starts a kafka consumer per partition for this topic
func (cons Kafka) Consume(threads *sync.WaitGroup) {
	err := cons.startConsumers()
	if err != nil {
		Log.Error.Print("Kafka client error:", err)
		return
	}

	cons.quit = false
	dumpIndexTicker := time.NewTicker(cons.offsetTimeout)

	defer func() {
		cons.quit = true
		cons.client.Close()
		cons.dumpIndex()
		cons.markAsDone()
	}()

	cons.markAsActive(threads)

	for cons.IsActive() {
		select {
		case command := <-cons.control:
			if command == shared.ConsumerControlStop {
				return // ### return ###
			}
		case <-dumpIndexTicker.C:
			if cons.offsetFile != "" {
				cons.dumpIndex()
			}
		}
	}
}
