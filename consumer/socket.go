// Copyright 2015-2017 trivago GmbH
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
	"github.com/trivago/tgo"
	"github.com/trivago/tgo/tio"
	"github.com/trivago/tgo/tnet"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	socketBufferGrowSize = 256
)

// Socket consumer plugin
//
// The socket consumer reads messages directly as-is from a given socket.
// Messages are separated from the stream by using a specific partitioner method.
//
// Configuration example
//
//  - "consumer.Socket":
//    Address: ":5880"
//    Permissions: "0770"
//    Acknowledge: ""
//    Partitioner: "delimiter"
//    Delimiter: "\n"
//    Offset: 0
//    Size: 1
//    ReconnectAfterSec: 2
//    AckTimoutSec: 2
//    ReadTimeoutSec: 5
//
// Address defines the protocol, host and port or socket to bind to.
// This can either be any ip address and port like "localhost:5880" or a file
// like "unix:///var/gollum.socket". By default this is set to ":5880".
//
// Permissions sets the file permissions for "unix://" based connections as an
// four digit octal number string. By default this is set to "0770".
//
// Acknowledge can be set to a non-empty value to inform the writer on success
// or error. On success the given string is send. Any error will close the
// connection. This setting is disabled by default, i.e. set to "".
// If Acknowledge is enabled and a IP-Address is given to Address, TCP is
// used to open the connection, otherwise UDP is used.
// If an error occurs during write "NOT <Acknowledge>" is returned.
//
// Partitioner defines the algorithm used to read messages from the router.
// By default this is set to "delimiter".
// * "delimiter" separates messages by looking for a delimiter string.
//   The delimiter is removed from the message.
// * "ascii" reads an ASCII number at a given offset until a given delimiter is found.
//   Everything to the right of and including the delimiter is removed from the message.
// * "binary" reads a binary number at a given offset and size.
// * "binary_le" is an alias for "binary".
// * "binary_be" is the same as "binary" but uses big endian encoding.
// * "fixed" assumes fixed size messages.
//
// Delimiter defines the delimiter used by the text and delimiter partitioner.
// By default this is set to "\n".
//
// Offset defines the offset used by the binary and text partitioner.
// By default this is set to 0. This setting is ignored by the fixed partitioner.
//
// Size defines the size in bytes used by the binary or fixed partitioner.
// For binary this can be set to 1,2,4 or 8. By default 4 is chosen.
// For fixed this defines the size of a message. By default 1 is chosen.
//
// ReconnectAfterSec defines the number of seconds to wait before a connection
// is tried to be reopened again. By default this is set to 2.
//
// AckTimoutSec defines the number of seconds waited for an acknowledge to
// succeed. Set to 2 by default.
//
// ReadTimoutSec defines the number of seconds that waited for data to be
// received. Set to 5 by default.
//
// RemoveOldSocket toggles removing exisiting files with the same name as the
// socket (unix://<path>) prior to connecting. Enabled by default.
type Socket struct {
	core.SimpleConsumer `gollumdoc:"embed_type"`
	listen              io.Closer
	clientLock          *sync.Mutex
	clients             *list.List
	protocol            string
	address             string
	acknowledge         string        `config:"Acknowledge"`
	delimiter           string        `config:"Delimiter" default:"\n"`
	reconnectTime       time.Duration `config:"ReconnectAfterSec" default:"2" metric:"sec"`
	ackTimeout          time.Duration `config:"AckTimoutSec" default:"2" metric:"sec"`
	readTimeout         time.Duration `config:"ReadTimoutSec" default:"5" metric:"sec"`
	fileFlags           os.FileMode   `config:"Permissions" default:"0770"`
	clearSocket         bool          `config:"RemoveOldSocket" default:"true"`
	offset              int           `config:"Offset"`
	flags               tio.BufferedReaderFlags
}

func init() {
	core.TypeRegistry.Register(Socket{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *Socket) Configure(conf core.PluginConfigReader) {
	flags, err := strconv.ParseInt(conf.GetString("Permissions", "0770"), 8, 32)
	conf.Errors.Push(err)
	cons.fileFlags = os.FileMode(flags)

	cons.clients = list.New()
	cons.clientLock = new(sync.Mutex)
	cons.protocol, cons.address = tnet.ParseAddress(conf.GetString("Address", ":5880"), "tcp")

	if cons.protocol != "unix" {
		if cons.acknowledge != "" {
			cons.protocol = "tcp"
		} else {
			cons.protocol = "udp"
		}
	}

	cons.flags = 0

	partitioner := strings.ToLower(conf.GetString("Partitioner", "delimiter"))
	switch partitioner {
	case "binary_be":
		cons.flags |= tio.BufferedReaderFlagBigEndian
		fallthrough

	case "binary", "binary_le":
		cons.flags |= tio.BufferedReaderFlagEverything
		switch conf.GetInt("Size", 4) {
		case 1:
			cons.flags |= tio.BufferedReaderFlagMLE8
		case 2:
			cons.flags |= tio.BufferedReaderFlagMLE16
		case 4:
			cons.flags |= tio.BufferedReaderFlagMLE32
		case 8:
			cons.flags |= tio.BufferedReaderFlagMLE64
		default:
			conf.Errors.Pushf("Size only supports the value 1,2,4 and 8")
		}

	case "fixed":
		cons.flags |= tio.BufferedReaderFlagMLEFixed
		cons.offset = int(conf.GetInt("Size", 1))

	case "ascii":
		cons.flags |= tio.BufferedReaderFlagMLE

	case "delimiter":
		// Nothing to add

	default:
		conf.Errors.Pushf("Unknown partitioner: %s", partitioner)
	}
}

func (cons *Socket) sendAck(conn net.Conn, success bool) error {
	if cons.acknowledge != "" {
		var err error
		conn.SetWriteDeadline(time.Now().Add(cons.ackTimeout))
		if success {
			_, err = fmt.Fprint(conn, cons.acknowledge)
		} else {
			_, err = fmt.Fprint(conn, "NOT "+cons.acknowledge)
		}
		return err
	}
	return nil
}

func (cons *Socket) processConnection(conn net.Conn) {
	cons.AddWorker()
	defer cons.WorkerDone()
	defer conn.Close()

	buffer := tio.NewBufferedReader(socketBufferGrowSize, cons.flags, cons.offset, cons.delimiter)

	for cons.IsActive() {
		conn.SetReadDeadline(time.Now().Add(cons.readTimeout))
		err := buffer.ReadAll(conn, cons.Enqueue)
		if err == nil {
			if err = cons.sendAck(conn, true); err == nil {
				continue // ### continue, all is well ###
			}
		}

		// Silently exit on disconnect/close
		if !cons.IsActive() || tnet.IsDisconnectedError(err) {
			return // ### return, connection closed ###
		}

		// Ignore timeout related errors
		if netErr, isNetErr := err.(net.Error); isNetErr && netErr.Timeout() {
			continue // ### return, ignore timeouts ###
		}

		cons.Logger.Error("Transfer failed: ", err)
		cons.sendAck(conn, false)

		// Parser errors do not drop the connection
		if err != tio.BufferDataInvalid {
			return // ### return, close connections ###
		}
	}
}

func (cons *Socket) processClientConnection(clientElement *list.Element) {
	defer func() {
		cons.clientLock.Lock()
		cons.clients.Remove(clientElement)
		cons.clientLock.Unlock()
	}()

	conn := clientElement.Value.(net.Conn)
	cons.processConnection(conn)
}

func (cons *Socket) udpAccept() {
	defer cons.WorkerDone()
	defer cons.closeConnection()
	addr, _ := net.ResolveUDPAddr(cons.protocol, cons.address)

	for cons.IsActive() {
		// (re)open a tcp connection
		for cons.listen == nil {
			if listener, err := net.ListenUDP(cons.protocol, addr); err == nil {
				cons.listen = listener
			} else {
				cons.Logger.Error("Connection error: ", err)
				time.Sleep(cons.reconnectTime)
			}
		}

		conn := cons.listen.(*net.UDPConn)
		cons.processConnection(conn)
		cons.listen = nil
	}
}

func (cons *Socket) tcpAccept() {
	defer cons.WorkerDone()
	defer cons.closeTCPConnection()

	for cons.IsActive() {
		// (re)open a tcp connection
		for cons.listen == nil {
			listener, err := net.Listen(cons.protocol, cons.address)
			if cons.protocol == "unix" && err == nil {
				err = os.Chmod(cons.address, cons.fileFlags)
			}

			if err == nil {
				cons.listen = listener
			} else {
				cons.Logger.Error("Connection error: ", err)

				// Clear socket if necessary
				if cons.protocol == "unix" && cons.clearSocket {
					// Try to create the socket file to check if it exists
					if socketFile, err := os.Create(cons.address); os.IsExist(err) {
						cons.Logger.Warning("Found existing socket ", cons.address, ". Removing.")

						if err := os.Remove(cons.address); err != nil {
							cons.Logger.Warning("Found existing socket ", cons.address, ". Removing.")
						} else {
							cons.Logger.Errorf("Socket %s cleared", cons.address)
						}
					} else {
						cons.Logger.Errorf("Existing socket %s was removed by third party", cons.address)
						socketFile.Close()
						if err := os.Remove(cons.address); err != nil {
							cons.Logger.Error("Could not remove test socket ", cons.address)
						}
					}
				}

				time.Sleep(cons.reconnectTime)
			}
		}

		//cons.Logger.Info("Listening to open: ", cons.address)
		listener := cons.listen.(net.Listener)
		if client, err := listener.Accept(); err != nil {
			// Trigger full reconnect (suppress errors during shutdown)
			if cons.IsActive() {
				cons.Logger.Error("Accept failed: ", err)
			}
			cons.closeTCPConnection()
		} else {
			// Handle client connection
			cons.clientLock.Lock()
			element := cons.clients.PushBack(client)
			cons.clientLock.Unlock()
			go cons.processClientConnection(element)
		}
	}
}

func (cons *Socket) closeConnection() {
	if cons.listen != nil {
		cons.listen.Close()
		cons.listen = nil
	}
}

func (cons *Socket) closeTCPConnection() {
	cons.closeConnection()
	cons.closeAllClients()
}

func (cons *Socket) closeAllClients() {
	cons.clientLock.Lock()
	defer cons.clientLock.Unlock()

	for cons.clients.Len() > 0 {
		client := cons.clients.Front()
		conn := client.Value.(net.Conn)
		cons.clients.Remove(client)
		conn.Close()
	}
}

// Consume listens to a given socket.
func (cons *Socket) Consume(workers *sync.WaitGroup) {
	cons.AddMainWorker(workers)

	if cons.protocol == "udp" {
		go tgo.WithRecoverShutdown(cons.udpAccept)
		defer cons.closeConnection()
	} else {
		go tgo.WithRecoverShutdown(cons.tcpAccept)
		defer cons.closeTCPConnection()
	}

	cons.ControlLoop()
}
