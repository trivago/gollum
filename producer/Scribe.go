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
//   Stream:
//     - "console"
//     - "_GOLLUM_"
//   Category:
//     "console" : "default"
//     "_GOLLUM_"  : "default"
//
// Host and Port should be clear
// Stream is a standard configuration field and denotes all the streams this
// producer will listen to.
// Category maps a stream to a specific scribe category.
type Scribe struct {
	standardProducer
	scribe    *scribe.ScribeClient
	transport *thrift.TFramedTransport
	socket    *thrift.TSocket
	category  map[string]string
}

var ScribeClassID = shared.Plugin.Register(Scribe{})

func (prod Scribe) Create(conf shared.PluginConfig) (shared.Producer, error) {

	err := prod.configureStandardProducer(conf)
	if err != nil {
		return nil, err
	}

	host, hostSet := conf.Settings["Host"]
	port, portSet := conf.Settings["Port"]
	category, categorySet := conf.Settings["Category"]

	if !hostSet {
		host = "localhost"
	}

	if !portSet {
		port = 1463
	}

	prod.category = make(map[string]string, 0)

	if !categorySet {
		for _, stream := range conf.Stream {
			prod.category[stream] = "default"
		}
	} else {
		categoryMap := category.(map[interface{}]interface{})
		for stream, category := range categoryMap {
			prod.category[stream.(string)] = category.(string)
		}
	}

	// Initialize scribe connection

	prod.socket, err = thrift.NewTSocket(host.(string) + ":" + strconv.Itoa(port.(int)))
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

func (prod Scribe) Produce() {
	defer func() {
		prod.transport.Close()
		prod.socket.Close()
		prod.response <- shared.ProducerControlResponseDone
	}()

	var category string

	for {
		select {
		case message := <-prod.messages:
			wildcardCategory, categorySet := prod.category["*"]

			if categorySet {
				category = wildcardCategory
			} else {
				category, categorySet = prod.category[message.Stream]
				if !categorySet {
					category = "default"
				}
			}

			logEntry := scribe.LogEntry{
				Category: category,
				Message:  message.Format(),
			}

			scribeMessages := make([]*scribe.LogEntry, 1)
			scribeMessages[0] = &logEntry
			result, err := prod.scribe.Log(scribeMessages)
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
			}

		case command := <-prod.control:
			if command == shared.ProducerControlStop {
				//fmt.Println("prod producer recieved stop")
				return // ### return, done ###
			}
		default:
			// Don't block
		}
	}
}
