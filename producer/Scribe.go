package producer

import (
	"fmt"
	"github.com/artyom/scribe"
	"github.com/artyom/thrift"
	"gollum/shared"
	"strconv"
)

type Scribe struct {
	standardProducer
	scribe    *scribe.ScribeClient
	transport *thrift.TFramedTransport
	socket    *thrift.TSocket
	category  string
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

	if !categorySet {
		category = "default"
	}

	prod.socket, err = thrift.NewTSocket(host.(string) + ":" + strconv.Itoa(port.(int)))
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	prod.transport = thrift.NewTFramedTransport(prod.socket)
	err = prod.transport.Open()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	protocolFactory := thrift.NewTBinaryProtocolFactory(false, false)

	prod.scribe = scribe.NewScribeClientFactory(prod.transport, protocolFactory)
	prod.category = category.(string)
	return prod, nil
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
			logEntry := scribe.LogEntry{
				Category: prod.category,
				Message:  message.Format(),
			}

			scribeMessages := make([]*scribe.LogEntry, 1)
			scribeMessages[0] = &logEntry
			result, err := prod.scribe.Log(scribeMessages)
			if err != nil {
				fmt.Println("scribe error", result, ":", err)
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
