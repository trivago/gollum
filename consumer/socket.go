// Copyright 2015-2018 trivago N.V.
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
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo"
	"github.com/trivago/tgo/tio"
	"github.com/trivago/tgo/tnet"
)

const (
	socketBufferGrowSize = 256
)

// Socket consumer plugin
//
// The socket consumer reads messages as-is from a given network or filesystem
// socket. Messages are separated from the stream by using a specific partitioner
// method.
//
// Parameters
//
// - Address: This value defines the protocol, host and port or socket to bind to.
// This can either be any ip address and port like "localhost:5880" or a file
// like "unix:///var/gollum.socket". Valid protocols can be derived from the
// golang net package documentation. Common values are "udp", "tcp" and "unix".
// By default this parameter is set to "tcp://0.0.0.0:5880".
//
// - Permissions: This value sets the filesystem permissions for UNIX domain
// sockets as a four-digit octal number.
// By default this parameter is set to "0770".
//
// - Acknowledge: This value can be set to a non-empty value to inform the writer
// that data has been accepted. On success, the given string is sent. Any error
// will close the connection. Acknowledge does not work with UDP based sockets.
// By default this parameter is set to "".
//
// - Partitioner: This value defines the algorithm used to read messages from the
// router. By default this is set to "delimiter". The following options are available:
//  - "delimiter": Separates messages by looking for a delimiter string.
//  The delimiter is removed from the message.
//  - "ascii": Reads an ASCII number at a given offset until a given delimiter is found.
//  Everything to the right of and including the delimiter is removed from the message.
//  - "binary": Reads a binary number at a given offset and size.
//  - "binary_le": An alias for "binary".
//  - "binary_be": The same as "binary" but uses big endian encoding.
//  - "fixed": Assumes fixed size messages.
//
// - Delimiter: This value defines the delimiter used by the text and delimiter
// partitioners.
// By default this parameter is set to "\n".
//
// - Offset: This value defines the offset used by the binary and text partitioners.
// This setting is ignored by the fixed partitioner.
// By default this parameter is set to "0".
//
// - Size: This value defines the size in bytes used by the binary and fixed
// partitioners. For binary, this can be set to 1,2,4 or 8. The default value
// is 4. For fixed , this defines the size of a message. By default this parameter
// is set to "1".
//
// - ReconnectAfterSec: This value defines the number of seconds to wait before a
// connection is retried.
// By default this parameter is set to "2".
//
// - AckTimeoutSec: This value defines the number of seconds to wait for acknowledges
// to succeed.
// By default this parameter is set to "1".
//
// - ReadTimeoutSec: This value defines the number of seconds to wait for data
// to be received. This setting affects the maximum shutdown duration of this consumer.
// By default this parameter is set to "2".
//
// - RemoveOldSocket: If set to true, any existing file with the same name as the
// socket (unix://<path>) is removed prior to connecting.
// By default this parameter is set to "true".
//
//
// Examples
//
// This example open a socket and expect messages with a fixed length of 256 bytes:
//
//  socketIn:
//    Type: consumer.Socket
//    Address: unix:///var/gollum.socket
//    Partitioner: fixed
//    Size: 256
//
type Socket struct {
	core.SimpleConsumer `gollumdoc:"embed_type"`
	listener            io.Closer
	protocol            string
	address             string
	acknowledge         string        `config:"Acknowledge" default:""`
	delimiter           string        `config:"Delimiter" default:"\n"`
	reconnectTime       time.Duration `config:"ReconnectAfterSec" default:"2" metric:"sec"`
	ackTimeout          time.Duration `config:"AckTimeoutSec" default:"1" metric:"sec"`
	readTimeout         time.Duration `config:"ReadTimeoutSec" default:"2" metric:"sec"`
	fileFlags           os.FileMode   `config:"Permissions" default:"0770"`
	offset              int           `config:"Offset" default:"0"`
	flags               tio.BufferedReaderFlags
	clearSocket         bool `config:"RemoveOldSocket" default:"true"`
}

func init() {
	core.TypeRegistry.Register(Socket{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *Socket) Configure(conf core.PluginConfigReader) {
	if len(cons.acknowledge) > 0 && cons.protocol == "udp" {
		conf.Errors.Pushf("UDP sockets do not support acknowledgment.")
	}

	address := conf.GetString("Address", "tcp://0.0.0.0:5880")
	cons.protocol, cons.address = tnet.ParseAddress(address, "tcp")
	cons.flags = 0

	partitioner := conf.GetString("Partitioner", "delimiter")
	switch strings.ToLower(partitioner) {
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

func (cons *Socket) listenUDP() {
	defer cons.WorkerDone()
	var (
		err    error
		socket net.Conn
	)

	addr, _ := net.ResolveUDPAddr(cons.protocol, cons.address)
	for cons.IsActive() {
		// (re)open a UDP connection
		for cons.listener == nil {
			if !cons.IsActive() {
				return // return, abort
			}

			socket, err = net.ListenUDP(cons.protocol, addr)
			if err == nil {
				cons.listener = socket
				cons.Logger.Debugf("Listening to %s", cons.address)
				break // break, listening
			}

			cons.Logger.WithError(err).Errorf("Failed to listen to %s", cons.address)
			time.Sleep(cons.reconnectTime)
		}

		cons.readFromConnection(socket, nil)
		cons.closeListener()
	}
}

func (cons *Socket) listen() {
	defer cons.WorkerDone()

	var (
		err        error
		socket     net.Listener
		forceClose *bool
	)

	for cons.IsActive() {
		// (re)open a tcp connection
		for cons.listener == nil {
			if !cons.IsActive() {
				return // return, abort
			}

			socket, err = net.Listen(cons.protocol, cons.address)
			if err == nil && cons.protocol == "unix" {
				err = os.Chmod(cons.address, cons.fileFlags)
			}

			if err == nil {
				cons.listener = socket
				forceClose = new(bool) // new trigger for all clients from this listener
				cons.Logger.Debugf("Listening to %s", cons.address)
				break // break, listening
			}

			cons.Logger.WithError(err).Errorf("Failed to listen to %s", cons.address)
			if cons.clearSocket && cons.protocol == "unix" {
				cons.tryRemoveUnixSocket()
			}
			time.Sleep(cons.reconnectTime)
		}

		conn, err := socket.Accept()
		if err == nil {
			cons.Logger.Debugf("New client connection to %s for %s", conn.RemoteAddr(), cons.address)
			cons.AddWorker()
			go cons.readFromClientConnection(conn, forceClose)
			continue // continue, accepted
		}

		if !cons.IsActive() {
			return // return, shutdown
		}

		// Trigger full reconnect (suppress errors during shutdown)
		cons.Logger.WithError(err).Errorf("Socket accept failed for %s ", cons.address)
		cons.closeListener()
		*forceClose = true
	}
}

func (cons *Socket) readFromClientConnection(conn net.Conn, forceClose *bool) {
	defer func() {
		conn.Close()
		cons.Logger.Debugf("Closed client connection to %s on %s", conn.RemoteAddr(), cons.address)
		cons.WorkerDone()
	}()
	cons.readFromConnection(conn, forceClose)
}

func (cons *Socket) readFromConnection(conn net.Conn, forceClose *bool) {
	buffer := tio.NewBufferedReader(socketBufferGrowSize, cons.flags, cons.offset, cons.delimiter)

	for cons.IsActive() && (forceClose == nil || !*forceClose) {
		// Read from connection
		// Time out in regular intervals so we can stop the loop on shutdown
		conn.SetReadDeadline(time.Now().Add(cons.readTimeout))
		if err := buffer.ReadAll(conn, cons.Enqueue); err != nil {
			netErr, isNetErr := err.(net.Error)
			switch {
			case !cons.IsActive():
				return

			case tnet.IsDisconnectedError(err):
				cons.Logger.Infof("Client %s closed connection", conn.RemoteAddr())
				return // return, closed

			case err == tio.BufferDataInvalid:
				cons.Logger.Infof("Invalid buffer data returnd from %s", conn.RemoteAddr())
				return // return, invalid data

			case isNetErr && netErr.Timeout():
				//cons.Logger.Infof("Read from %s timed out", conn.RemoteAddr())
				continue

			default:
				remote := conn.RemoteAddr()
				if remote == nil {
					cons.Logger.WithError(err).Errorf("Failed to read from %s", cons.address)
				} else {
					cons.Logger.WithError(err).Errorf("Failed to read from %s on %s", remote, cons.address)
				}
				continue
			}
		}

		// Send ack if required
		if err := cons.sendACK(conn); err != nil {
			cons.Logger.WithError(err).Errorf("Failed to send ack to %s", conn.RemoteAddr())
		}
	}
}

func (cons *Socket) sendACK(conn net.Conn) error {
	if len(cons.acknowledge) == 0 || cons.protocol == "udp" {
		return nil
	}
	conn.SetWriteDeadline(time.Now().Add(cons.ackTimeout))
	_, err := fmt.Fprint(conn, cons.acknowledge)
	return err
}

func (cons *Socket) closeListener() {
	if cons.listener == nil {
		return
	}
	cons.Logger.Debugf("Closing socket %s", cons.address)
	cons.listener.Close()
	cons.listener = nil
}

func (cons *Socket) tryRemoveUnixSocket() {
	// Try to create the socket file to check if it exists
	socketFile, err := os.Create(cons.address)
	if os.IsExist(err) {
		cons.Logger.Warningf("Found existing socket %s. Trying to remove it", cons.address)
		if err := os.Remove(cons.address); err != nil {
			cons.Logger.WithError(err).Errorf("Failed to remove %s", cons.address)
			return
		}

		cons.Logger.Warningf("Removed %s", cons.address)
		return
	}

	socketFile.Close()
	if err := os.Remove(cons.address); err != nil {
		cons.Logger.WithError(err).Errorf("Could not remove test file %s", cons.address)
	}
}

// Consume listens to a given socket.
func (cons *Socket) Consume(workers *sync.WaitGroup) {
	cons.AddMainWorker(workers)
	defer cons.closeListener()

	if cons.protocol == "udp" {
		go tgo.WithRecoverShutdown(cons.listenUDP)
	} else {
		go tgo.WithRecoverShutdown(cons.listen)
	}

	cons.ControlLoop()
}
