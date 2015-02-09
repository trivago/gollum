package consumer

import (
	"fmt"
	"github.com/trivago/gollum/shared"
	"io"
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
//
// Acknowledge can be set to true to inform the writer on success or error.
// On success "OK\n" is send. Any error will close the connection.
// This setting is disabled by default.
type Socket struct {
	standardConsumer
	listen      net.Listener
	protocol    string
	address     string
	delimiter   string
	runlength   bool
	quit        bool
	acknowledge bool
}

func init() {
	shared.RuntimeType.Register(Socket{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *Socket) Configure(conf shared.PluginConfig) error {
	err := cons.standardConsumer.Configure(conf)
	if err != nil {
		return err
	}

	escapeChars := strings.NewReplacer("\\n", "\n", "\\r", "\r", "\\t", "\t")

	cons.runlength = conf.GetBool("Runlength", false)
	cons.delimiter = escapeChars.Replace(conf.GetString("Delimiter", "\n"))
	cons.address = conf.GetString("Address", ":5880")
	cons.protocol = "tcp"
	cons.acknowledge = conf.GetBool("Acknowledge", false)

	if strings.HasPrefix(cons.address, fileSocketPrefix) {
		cons.address = cons.address[len(fileSocketPrefix):]
		cons.protocol = "unix"
	}

	cons.quit = false
	return err
}

func (cons *Socket) readFromConnection(conn net.Conn) {
	defer conn.Close()

	var err error
	buffer := shared.CreateBufferedReader(socketBufferGrowSize, cons.postMessageFromSlice)

	for !cons.quit {
		// Read from stream
		if cons.runlength {
			err = buffer.ReadRLE(conn)
		} else {
			err = buffer.Read(conn, cons.delimiter)
		}

		if err == nil || err == io.EOF {
			if cons.acknowledge {
				fmt.Fprint(conn, "OK")
			}
		} else {
			if !cons.quit {
				shared.Log.Error.Print("Socket read failed:", err)
			}
			break // ### break, close connection ###
		}
	}
}

func (cons *Socket) accept(threads *sync.WaitGroup) {
	for !cons.quit {

		client, err := cons.listen.Accept()
		if err != nil {
			if !cons.quit {
				shared.Log.Error.Print("Socket listen failed:", err)
			}
			break // ### break ###
		}

		go cons.readFromConnection(client)
	}

	cons.markAsDone()
}

// Consume listens to a given socket. Messages are expected to be separated by
// either \n or \r\n.
func (cons Socket) Consume(threads *sync.WaitGroup) {
	// Listen to socket

	var err error
	cons.listen, err = net.Listen(cons.protocol, cons.address)
	if err != nil {
		shared.Log.Error.Print("Socket connection error: ", err)
		return
	}

	go cons.accept(threads)
	defer func() {
		cons.quit = true
		cons.listen.Close()
	}()

	cons.defaultControlLoop(threads)
}
