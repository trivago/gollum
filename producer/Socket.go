package producer

import (
	"fmt"
	"gollum/shared"
	"net"
	"strings"
)

var fileSocketPrefix = "unix://"

// Console producer plugin
// Configuration example
//
// - "producer.Socket":
//   Enable: true
//   Address: "unix:///var/gollum.socket"
//
// Address stores the identifier to connect to.
// This can either be any ip address and port like "localhost:5880" or a file
// like "unix:///var/gollum.socket". By default this is set to ":5880".
type Socket struct {
	standardProducer
	connection net.Conn
	protocol   string
	address    string
}

func init() {
	shared.Plugin.Register(Socket{})
}

func (prod Socket) Create(conf shared.PluginConfig) (shared.Producer, error) {
	err := prod.configureStandardProducer(conf)
	if err != nil {
		return nil, err
	}

	address, addressSet := conf.Settings["Address"]

	prod.protocol = "tcp"
	prod.address = ":5880"

	if addressSet {
		prod.address = address.(string)
		if strings.HasPrefix(prod.address, fileSocketPrefix) {
			prod.address = prod.address[len(fileSocketPrefix):]
			prod.protocol = "unix"
		}
	}

	prod.connection, err = net.Dial(prod.protocol, prod.address)
	if err != nil {
		shared.Log.Error("Socket connection error:", err)
	}

	return prod, nil
}

func (prod Socket) Produce() {
	defer func() {
		if prod.connection != nil {
			prod.connection.Close()
		}
		prod.response <- shared.ProducerControlResponseDone
	}()

	for {
		select {
		case message := <-prod.messages:
			if prod.connection == nil {
				var err error
				prod.connection, err = net.Dial(prod.protocol, prod.address)

				if err != nil {
					shared.Log.Error("Socket connection error:", err)
				}
			}

			if prod.connection != nil {
				_, err := fmt.Fprintln(prod.connection, message.Format(prod.forward))

				if err != nil {
					shared.Log.Error("Socket error:", err)
					prod.connection.Close()
					prod.connection = nil
				}
			}

		case command := <-prod.control:
			if command == shared.ProducerControlStop {
				return // ### return, done ###
			}
		default:
			// Don't block
		}
	}
}
