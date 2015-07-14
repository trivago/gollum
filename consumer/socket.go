// Copyright 2015 trivago GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package consumer

import (
	"container/list"
	"fmt"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	"io"
	"net"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	socketBufferGrowSize = 256
)

// Socket consumer plugin
// Configuration example
//
//   - "consumer.Socket":
//     Enable: true
//     Address: "unix:///var/gollum.socket"
//     Acknowledge: "ACK\n"
//     Partitioner: "ascii"
//     Delimiter: ":"
//     Offset: 1
//
// The socket consumer reads messages directly as-is from a given socket.
// Messages are separated from the stream by using a specific paritioner method.
//
// Address stores the identifier to bind to.
// This can either be any ip address and port like "localhost:5880" or a file
// like "unix:///var/gollum.socket". By default this is set to ":5880".
//
// Acknowledge can be set to a non-empty value to inform the writer on success
// or error. On success the given string is send. Any error will close the
// connection. This setting is disabled by default, i.e. set to "".
// If Acknowledge is enabled and a IP-Address is given to Address, TCP is
// used to open the connection, otherwise UDP is used.
//
// Partitioner defines the algorithm used to read messages from the stream.
// By default this is set to "delimiter".
//  - "delimiter" separates messages by looking for a delimiter string. The
//    delimiter is removed from the message.
//  - "ascii" reads an ASCII encoded number at a given offset until a given
//    delimiter is found. Everything to the right of and including the delimiter
//    is removed from the message.
//  - "binary" reads a binary number at a given offset and size
//  - "binary_le" is an alias for "binary"
//  - "binary_be" is the same as "binary" but uses big endian encoding
//  - "fixed" assumes fixed size messages
//
// Delimiter defines the delimiter used by the text and delimiter partitioner.
// By default this is set to "\n".
//
// Offset defines the offset used by the binary and text paritioner.
// By default this is set to 0. This setting is ignored by the fixed partitioner.
//
// Size defines the size in bytes used by the binary or fixed partitioner.
// For binary this can be set to 1,2,4 or 8. By default 4 is chosen.
// For fixed this defines the size of a message. By default 1 is chosen.
type Socket struct {
	core.ConsumerBase
	listen      io.Closer
	clientLock  *sync.Mutex
	clients     *list.List
	protocol    string
	address     string
	delimiter   string
	acknowledge string
	flags       shared.BufferedReaderFlags
	offset      int
	quit        bool
}

func init() {
	shared.RuntimeType.Register(Socket{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *Socket) Configure(conf core.PluginConfig) error {
	err := cons.ConsumerBase.Configure(conf)
	if err != nil {
		return err
	}

	cons.clients = list.New()
	cons.clientLock = new(sync.Mutex)
	cons.acknowledge = shared.Unescape(conf.GetString("Acknowledge", ""))
	cons.address, cons.protocol = shared.ParseAddress(conf.GetString("Address", ":5880"))

	if cons.protocol != "unix" {
		if cons.acknowledge != "" {
			cons.protocol = "tcp"
		} else {
			cons.protocol = "udp"
		}
	}

	cons.delimiter = shared.Unescape(conf.GetString("Delimiter", "\n"))
	cons.offset = conf.GetInt("Offset", 0)
	cons.flags = 0

	partitioner := strings.ToLower(conf.GetString("Partitioner", "delimiter"))
	switch partitioner {
	case "binary_be":
		cons.flags |= shared.BufferedReaderFlagBigEndian
		fallthrough

	case "binary", "binary_le":
		cons.flags |= shared.BufferedReaderFlagEverything
		switch conf.GetInt("Size", 4) {
		case 1:
			cons.flags |= shared.BufferedReaderFlagMLE8
		case 2:
			cons.flags |= shared.BufferedReaderFlagMLE16
		case 4:
			cons.flags |= shared.BufferedReaderFlagMLE32
		case 8:
			cons.flags |= shared.BufferedReaderFlagMLE64
		default:
			return fmt.Errorf("Size only supports the value 1,2,4 and 8")
		}

	case "fixed":
		cons.flags |= shared.BufferedReaderFlagMLEFixed
		cons.offset = conf.GetInt("Size", 1)

	case "ascii":
		cons.flags |= shared.BufferedReaderFlagMLE

	case "delimiter":
		// Nothing to add

	default:
		return fmt.Errorf("Unknown partitioner: %s", partitioner)
	}

	cons.quit = false
	return err
}

func (cons *Socket) clientDisconnected(err error) bool {
	netErr, isNetErr := err.(*net.OpError)
	if isNetErr {

		errno, isErrno := netErr.Err.(syscall.Errno)
		if isErrno {
			switch errno {
			default:
			case syscall.ECONNRESET:
				return true // ### return, close connection ###
			}
		}
	}

	return false
}

func (cons *Socket) processClientConnection(clientElement *list.Element) {
	defer shared.RecoverShutdown()
	conn := clientElement.Value.(net.Conn)

	defer func() {
		cons.clientLock.Lock()
		defer cons.clientLock.Unlock()

		cons.clients.Remove(clientElement)
		conn.Close()
		cons.WorkerDone()
	}()

	cons.AddWorker()
	cons.processConnection(conn)
}

func (cons *Socket) processConnection(conn net.Conn) {
	conn.SetDeadline(time.Time{})
	buffer := shared.NewBufferedReader(socketBufferGrowSize, cons.flags, cons.offset, cons.delimiter)

	for !cons.quit {
		err := buffer.ReadAll(conn, cons.Enqueue)

		// Handle errors
		if err != nil && err != io.EOF {
			if cons.quit || cons.clientDisconnected(err) {
				return // ### return, connection closed ###
			}

			Log.Error.Print("Socket read failed: ", err)
			continue // ### continue, keep open, try again ###
		}

		// Send ack if everything was ok
		if cons.acknowledge != "" {
			fmt.Fprint(conn, cons.acknowledge)
		}
	}
}

func (cons *Socket) udpAccept() {
	conn := cons.listen.(*net.UDPConn)
	cons.processConnection(conn)
}

func (cons *Socket) tcpAccept() {
	defer cons.WorkerDone()

	listener := cons.listen.(net.Listener)
	for !cons.quit {
		client, err := listener.Accept()
		if err != nil {
			if !cons.quit {
				Log.Error.Print("Socket listen failed: ", err)
			}
			break // ### break ###
		}

		cons.clientLock.Lock()
		element := cons.clients.PushBack(client)
		go cons.processClientConnection(element)
		cons.clientLock.Unlock()
	}

	cons.clientLock.Lock()
	defer cons.clientLock.Unlock()
	for iter := cons.clients.Front(); iter != nil; iter = iter.Next() {
		client := iter.Value.(net.Conn)
		client.Close()
	}
}

// Consume listens to a given socket.
func (cons *Socket) Consume(workers *sync.WaitGroup) {
	var err error
	var listen func()

	cons.quit = false
	if cons.protocol == "udp" {
		addr, _ := net.ResolveUDPAddr(cons.protocol, cons.address)
		if cons.listen, err = net.ListenUDP(cons.protocol, addr); err != nil {
			Log.Error.Print("Socket connection error: ", err)
			return
		}
		listen = cons.udpAccept
	} else {
		if cons.listen, err = net.Listen(cons.protocol, cons.address); err != nil {
			Log.Error.Print("Socket connection error: ", err)
			return
		}
		listen = cons.tcpAccept
	}

	go func() {
		defer shared.RecoverShutdown()
		cons.AddMainWorker(workers)
		listen()
	}()

	defer func() {
		cons.quit = true
		cons.listen.Close()
	}()

	cons.DefaultControlLoop(nil)
}
