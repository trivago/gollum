package consumer

import (
	"github.com/trivago/gollum/shared"
	"net"
	"strings"
	"sync"
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

func (cons Socket) Create(conf shared.PluginConfig, pool *shared.BytePool) (shared.Consumer, error) {
	err := cons.configureStandardConsumer(conf, pool)
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

		endIdx := offset + size
		startIdx := 0

		for i := offset; i < endIdx; i++ {
			if buffer[i] == '\n' {
				messageEndIdx := i
				if i > 0 && buffer[i-1] == '\r' {
					messageEndIdx-- // ...\r\n
				}

				cons.postMessageFromSlice(buffer[startIdx:messageEndIdx])
				startIdx = i + 1
			}
		}

		// Manage the buffer remains

		if startIdx == 0 {
			// If we did not move at all continue reading. If we don't have any
			// space left, resize the buffer by 1KB

			bufferSize := len(buffer)
			if endIdx == bufferSize {
				temp := buffer
				buffer = make([]byte, bufferSize+socketBufferGrowSize)
				copy(buffer, temp)
			}
			offset = endIdx

		} else if startIdx != endIdx {
			// If we did move but there are remains left in the buffer move them
			// to the start of the buffer and read again

			copy(buffer, buffer[startIdx:endIdx])
			offset = endIdx - startIdx
		} else {
			// Everything was written

			offset = 0
		}
	}
}

func (cons Socket) accept(quit *atomic.Value, threads *sync.WaitGroup) {
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

	threads.Done()
}

func (cons Socket) Consume(threads *sync.WaitGroup) {
	var quit atomic.Value
	quit.Store(false)
	threads.Add(1)

	go cons.accept(&quit, threads)

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
