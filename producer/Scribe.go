package producer

import (
	"github.com/artyom/scribe"
	"github.com/artyom/thrift"
	"gollum/shared"
	"strconv"
)

// Scribe producer plugin
// Configuration example
//
// - "producer.Scribe":
//   Enable: true
//   Host: "192.168.222.30"
//   Port: 1463
//   BatchSize: 4096
//   BatchSizeThreshold: 16777216
//   BatchTimeoutSec: 2
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
// BatchSize defines the number of bytes to be buffered before they are written
// to scribe. By default this is set to 8KB.
//
// BatchSizeThreshold defines the maximum number of bytes to buffer before
// messages get dropped. If a message crosses the threshold it is still buffered
// but additional messages will be dropped. By default this is set to 8MB.
//
// BatchTimeoutSec defines the maximum number of seconds to wait after the last
// message arrived before a batch is flushed automatically. By default this is
// set to 5.
type Scribe struct {
	standardProducer
	scribe          *scribe.ScribeClient
	transport       *thrift.TFramedTransport
	socket          *thrift.TSocket
	batch           *scribeMessageBuffer
	category        map[shared.MessageStreamID]string
	batchSize       int
	batchTimeoutSec int
	defaultCategory string
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
	batchSizeThreshold := conf.GetInt("BatchSizeThreshold", 8388608)

	prod.category = make(map[shared.MessageStreamID]string, 0)
	prod.batchSize = conf.GetInt("BatchSize", 8192)
	prod.batchTimeoutSec = conf.GetInt("BatchTimeoutSec", 5)
	prod.batch = createScribeMessageBuffer(batchSizeThreshold)

	// Read stream to category mapping

	defaultMapping := make(map[interface{}]interface{})
	defaultMapping[shared.WildcardStreamID] = "default"

	categoryMap := conf.GetValue("Category", defaultMapping).(map[interface{}]interface{})
	for stream, category := range categoryMap {
		prod.category[shared.GetStreamID(stream.(string))] = category.(string)
	}

	prod.defaultCategory = "default"

	wildcardCategory, wildcardCategorySet := prod.category[shared.WildcardStreamID]
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

func (prod Scribe) send() {
	result, err := prod.scribe.Log(prod.batch.get())

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
		prod.batch.flush()
	}
}

func (prod Scribe) Produce() {
	defer func() {
		prod.transport.Close()
		prod.socket.Close()
		prod.response <- shared.ProducerControlResponseDone
	}()

	for {
		select {
		case message := <-prod.messages:
			category, exists := prod.category[message.StreamID]
			if !exists {
				category = prod.defaultCategory
			}

			prod.batch.appendAndRelease(message, category, prod.forward)
			if prod.batch.reachedSizeThreshold(prod.batchSize) {
				prod.send()
			}

		case command := <-prod.control:
			if command == shared.ProducerControlStop {
				//fmt.Println("prod producer recieved stop")
				return // ### return, done ###
			}

		default:
			if prod.batch.reachedTimeThreshold(prod.batchTimeoutSec) {
				prod.send()
			}
			// Don't block
		}
	}
}
