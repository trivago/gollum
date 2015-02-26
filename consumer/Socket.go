package consumer

import (
	"fmt"
	"github.com/trivago/gollum/log"
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
//   Acknowledge: true
//   Runlength: true
//   Sequence: true
//   Delimiter: "\n"
//
// Address stores the identifier to bind to.
// This can either be any ip address and port like "localhost:5880" or a file
// like "unix:///var/gollum.socket". By default this is set to ":5880".
//
// Runlength should be set to true if the incoming messages are formatted with
// the runlegth formatter, i.e. there is a "length:" prefix.
// This option is disabled by default.
//
// Sequence should be used if the message is prefixed by a sequence number, i.e.
// "sequence:" is prepended to the message.
// In case that Runlength is set, too the Runlength prefix is expected first.
// This option is disabled by default.
//
// Delimiter defines a string that marks the end of a message. If Runlength is
// set this string is ignored.
//
// Acknowledge can be set to true to inform the writer on success or error.
// On success "OK\n" is send. Any error will close the connection.
// This setting is disabled by default.
// If Acknowledge is set to true and a IP-Address is given to Address, TCP is
// used to open the connection, otherwise UDP is used.
type Socket struct {
	shared.ConsumerBase
	listen      net.Listener
	protocol    string
	address     string
	delimiter   string
	runlength   bool
	sequence    bool
	quit        bool
	acknowledge bool
}

func init() {
	shared.RuntimeType.Register(Socket{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *Socket) Configure(conf shared.PluginConfig) error {
	err := cons.ConsumerBase.Configure(conf)
	if err != nil {
		return err
	}

	escapeChars := strings.NewReplacer("\\n", "\n", "\\r", "\r", "\\t", "\t")

	cons.delimiter = escapeChars.Replace(conf.GetString("Delimiter", "\n"))
	cons.address = conf.GetString("Address", ":5880")
	cons.acknowledge = conf.GetBool("Acknowledge", false)
	cons.runlength = conf.GetBool("Runlength", false)
	cons.sequence = conf.GetBool("Sequence", false)

	if cons.acknowledge {
		cons.protocol = "tcp"
	} else {
		cons.protocol = "udp"
	}

	if strings.HasPrefix(cons.address, fileSocketPrefix) {
		cons.address = cons.address[len(fileSocketPrefix):]
		cons.protocol = "unix"
	}

	cons.quit = false
	return err
}

func (cons *Socket) readFromConnection(conn net.Conn) {
	defer conn.Close()
	var buffer shared.BufferedReader

	if cons.sequence {
		buffer = shared.NewBufferedReaderSequence(socketBufferGrowSize, cons.PostMessageFromSlice)
	} else {
		buffer = shared.NewBufferedReader(socketBufferGrowSize, cons.PostMessageFromSlice, 0)
	}

	for !cons.quit {
		// Read from stream

		var err error
		switch {
		case cons.runlength:
			err = buffer.ReadRLE(conn)
		default:
			err = buffer.Read(conn, cons.delimiter)
		}

		if err == nil || err == io.EOF {
			if cons.acknowledge && cons.protocol == "tcp" {
				fmt.Fprint(conn, "OK")
			}
		} else {
			if !cons.quit {
				Log.Error.Print("Socket read failed:", err)
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
				Log.Error.Print("Socket listen failed:", err)
			}
			break // ### break ###
		}

		go cons.readFromConnection(client)
	}

	cons.MarkAsDone()
}

// Consume listens to a given socket. Messages are expected to be separated by
// either \n or \r\n.
func (cons Socket) Consume(threads *sync.WaitGroup) {
	// Listen to socket
	var err error
	if cons.listen, err = net.Listen(cons.protocol, cons.address); err != nil {
		Log.Error.Print("Socket connection error: ", err)
		return
	}

	cons.quit = false
	go cons.accept(threads)

	defer func() {
		cons.quit = true
		cons.listen.Close()
	}()

	cons.DefaultControlLoop(threads, nil)
}
