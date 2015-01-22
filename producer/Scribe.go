package producer

import (
	"github.com/artyom/scribe"
	"github.com/artyom/thrift"
	"gollum/shared"
	"strconv"
	"time"
)

// Scribe producer plugin
// Configuration example
//
// - "producer.Scribe":
//   Enable: true
//   Host: "192.168.222.30"
//   Port: 1463
//   Stream:
//     - "console"
//     - "_GOLLUM_"
//   Category:
//     "console" : "default"
//     "_GOLLUM_"  : "default"
//
// Host and Port should be clear
//
// Category maps a stream to a specific scribe category. You can define the
// wildcard stream (*) here, too. All streams that do not have a specific
// mapping will go to this stream (including _GOLLUM_).
// If no category mappings are set all messages will be send to "default".
//
// BatchSize defines the number of messages to store before a send is triggered.
// If the connection is down the number of messages stored may exceed this
// count. By default this is set to 100.
//
// BatchThreshold defines the maximum number of messages to store before
// messages are dropped. By default this is set to 10000.
//
// BatchTimeoutSec defines the maximum number of seconds to wait after the last
// message arrived before a batch is flushed automatically. By default this is
// set to 5.
type Scribe struct {
	standardProducer
	scribe          *scribe.ScribeClient
	transport       *thrift.TFramedTransport
	socket          *thrift.TSocket
	category        map[string]string
	batchSize       int
	batchThreshold  int
	batchTimeoutSec int
	defaultCategory string
}

type scribeMessageBuffer struct {
	message     []*scribe.LogEntry
	count       int
	lastMessage time.Time
}

func init() {
	shared.Plugin.Register(Scribe{})
}

func (prod Scribe) Create(conf shared.PluginConfig) (shared.Producer, error) {

	err := prod.configureStandardProducer(conf)
	if err != nil {
		return nil, err
	}

	host := conf.GetString("Host", "localhost")
	port := conf.GetInt("Port", 1463)

	prod.category = make(map[string]string, 0)
	prod.batchSize = conf.GetInt("BatchSize", 100)
	prod.batchThreshold = conf.GetInt("BatchThreshold", 10000)
	prod.batchTimeoutSec = conf.GetInt("BatchTimeoutSec", 5)

	// Read stream to category mapping

	defaultMapping := make(map[interface{}]interface{})
	defaultMapping["*"] = "default"

	categoryMap := conf.GetValue("Category", defaultMapping).(map[interface{}]interface{})
	for stream, category := range categoryMap {
		prod.category[stream.(string)] = category.(string)
	}

	prod.defaultCategory = "default"

	wildcardCategory, wildcardCategorySet := prod.category["*"]
	if wildcardCategorySet {
		prod.defaultCategory = wildcardCategory
	}

	// Initialize scribe connection

	prod.socket, err = thrift.NewTSocket(host + ":" + strconv.Itoa(port))
	if err != nil {
		shared.Log.Error("Scribe socket error:", err)
		return nil, err
	}

	prod.transport = thrift.NewTFramedTransport(prod.socket)
	err = prod.transport.Open()
	if err != nil {
		shared.Log.Error("Scribe transport error:", err)
		return nil, err
	}

	protocolFactory := thrift.NewTBinaryProtocolFactory(false, false)

	prod.scribe = scribe.NewScribeClientFactory(prod.transport, protocolFactory)
	return prod, nil
}

func (prod Scribe) sendBatch(batch *scribeMessageBuffer) {
	batch.lastMessage = time.Now()
	result, err := prod.scribe.Log(batch.message[:batch.count])

	if err != nil {
		shared.Log.Error("Scribe log error", result, ":", err)

		// Try to reopen the connection
		if err.Error() == "EOF" {
			prod.transport.Close()
			err = prod.transport.Open()
			if err != nil {
				shared.Log.Error("Scribe connection error:", err)
			}
		}
	} else {
		batch.count = 0
	}
}

func (prod Scribe) post(batch *scribeMessageBuffer, stream string, text string) {
	if batch.count < prod.batchThreshold {

		logEntry := new(scribe.LogEntry)
		logEntry.Category = prod.defaultCategory
		logEntry.Message = text

		category, categorySet := prod.category[stream]
		if categorySet {
			logEntry.Category = category
		}

		batch.message[batch.count] = logEntry
		batch.count++
		batch.lastMessage = time.Now()

		if batch.count == prod.batchSize {
			prod.sendBatch(batch)
		}
	}
}

func (prod Scribe) Produce() {
	defer func() {
		prod.transport.Close()
		prod.socket.Close()
		prod.response <- shared.ProducerControlResponseDone
	}()

	batch := scribeMessageBuffer{
		message:     make([]*scribe.LogEntry, prod.batchThreshold),
		count:       0,
		lastMessage: time.Now(),
	}

	for {
		select {
		case message := <-prod.messages:
			prod.post(&batch, message.Stream, message.Format(prod.forward))

		case command := <-prod.control:
			if command == shared.ProducerControlStop {
				//fmt.Println("prod producer recieved stop")
				return // ### return, done ###
			}

		default:
			if batch.count > 0 && time.Since(batch.lastMessage).Seconds() > float64(prod.batchTimeoutSec) {
				prod.sendBatch(&batch)
			}
			// Don't block
		}
	}
}
