package consumer

import (
	"github.com/trivago/gollum/shared"
	"net"
	"strings"
	"sync"
)

var fileSocketPrefix = "unix://"

const (
	socketBufferGrowSize = 256
)

// Socket consumer plugin
// Configuration example
//
// - "consumer.Socket":
//   Enable: true
//   Address: "unix:///var/gollum.socket"
//
// Address stores the identifier to bind to.
// This can either be any ip address and port like "localhost:5880" or a file
// like "unix:///var/gollum.socket". By default this is set to ":5880".
type Socket struct {
	standardConsumer
	listen   net.Listener
	protocol string
	address  string
	quit     bool
}

func init() {
	shared.Plugin.Register(Socket{})
}

// Create creates a new consumer based on the current socket consumer.
func (cons Socket) Create(conf shared.PluginConfig) (shared.Consumer, error) {
	err := cons.configureStandardConsumer(conf)
	if err != nil {
		return nil, err
	}

	cons.address = conf.GetString("Address", ":5880")
	cons.protocol = "tcp"

	if strings.HasPrefix(cons.address, fileSocketPrefix) {
		cons.address = cons.address[len(fileSocketPrefix):]
		cons.protocol = "unix"
	}

	cons.quit = false
	return cons, err
}

func (cons *Socket) readFromConnection(conn net.Conn) {
	buffer := shared.CreateBufferedReader(socketBufferGrowSize, cons.postMessageFromSlice)

	for !cons.quit {
		// Read from stream
		err := buffer.Read(conn, "\n")
		if err != nil {
			if !cons.quit {
				shared.Log.Error("Socket read failed:", err)
			}

			return // ### return ###
		}
	}
}

func (cons *Socket) accept(threads *sync.WaitGroup) {
	for !cons.quit {

		client, err := cons.listen.Accept()
		if err != nil {
			if !cons.quit {
				shared.Log.Error("Socket listen failed:", err)
			}
			break // ### break ###
		}

		go cons.readFromConnection(client)
	}

	threads.Done()
}

// Consume listens to a given socket. Messages are expected to be separated by
// either \n or \r\n.
func (cons Socket) Consume(threads *sync.WaitGroup) {
	// Listen to socket

	var err error
	cons.listen, err = net.Listen(cons.protocol, cons.address)
	if err != nil {
		shared.Log.Error("Socket connection error: ", err)
		return
	}

	threads.Add(1)

	go cons.accept(threads)

	defer func() {
		cons.quit = true
		cons.listen.Close()
	}()

	// Wait for control statements

	for {
		command := <-cons.control
		if command == shared.ConsumerControlStop {
			return // ### return ###
		}
	}
}
