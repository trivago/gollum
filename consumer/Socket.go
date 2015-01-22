package consumer

import (
	"gollum/shared"
	"net"
	"strings"
	"sync/atomic"
)

var fileSocketPrefix = "unix://"

const (
	socketBufferGrowSize = 1024
)

// Console consumer plugin
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
	listen net.Listener
}

func init() {
	shared.Plugin.Register(Socket{})
}

func (cons Socket) Create(conf shared.PluginConfig) (shared.Consumer, error) {
	err := cons.configureStandardConsumer(conf)
	if err != nil {
		return nil, err
	}

	address := conf.GetString("Address", ":5880")
	protocol := "tcp"

	if strings.HasPrefix(address, fileSocketPrefix) {
		address = address[len(fileSocketPrefix):]
		protocol = "unix"
	}

	cons.listen, err = net.Listen(protocol, address)
	if err != nil {
		return nil, err
	}

	return cons, err
}

func (cons Socket) readFromConnection(conn net.Conn, quit *atomic.Value) {
	buffer := make([]byte, socketBufferGrowSize)
	offset := 0

	for !quit.Load().(bool) {
		// Read from stream

		size, err := conn.Read(buffer[offset:])
		if err != nil {
			if !quit.Load().(bool) {
				shared.Log.Error("Socket read failed:", err)
			}

			return // ### return ###
		}

		// Go through the stream and look for line breaks (delimiter)
		// Send one message per delimiter

		end := offset + size
		start := 0

		for i := offset; i < end; i++ {
			if buffer[i] == '\n' {
				var message string
				if i > 0 && buffer[i-1] == '\r' {
					message = string(buffer[start : i-1]) // ...\r\n
				} else {
					message = string(buffer[start:i]) // ...\n
				}

				// Telnet quit support

				if message == "quit" {
					conn.Close()
					return // ### return, close requested ###
				}

				cons.postMessage(message)
				start = i + 1
			}
		}

		// Manage the buffer remains

		if start == 0 {
			// If we did not move at all continue reading. If we don't have any
			// space left, resize the buffer by 1KB

			bufferSize := len(buffer)
			if end == bufferSize {
				temp := buffer
				buffer = make([]byte, bufferSize+socketBufferGrowSize)
				copy(buffer, temp)
			}
			offset = end

		} else if start != end {
			// If we did move but there are remains left in the buffer move them
			// to the start of the buffer and read again

			copy(buffer, buffer[start:end])
			offset = end - start
		} else {
			// Everything was written

			offset = 0
		}
	}
}

func (cons Socket) accept(quit *atomic.Value) {
	for !quit.Load().(bool) {

		client, err := cons.listen.Accept()
		if err != nil {
			if !quit.Load().(bool) {
				shared.Log.Error("Socket listen failed:", err)
			}
			break // ### break ###
		}

		go cons.readFromConnection(client, quit)
	}

	cons.response <- shared.ConsumerControlResponseDone
}

func (cons Socket) Consume() {
	var quit atomic.Value
	quit.Store(false)
	go cons.accept(&quit)

	defer func() {
		quit.Store(true)
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
