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
	"fmt"
	"github.com/trivago/gollum/log"
	"github.com/trivago/gollum/shared"
	"io"
	"net"
	"strings"
	"sync"
	"syscall"
)

var fileSocketPrefix = "unix://"

const (
	socketBufferGrowSize = 256
)

// Socket consumer plugin
// Configuration example
//
//   - "consumer.Socket":
//     Enable: true
//     Address: "unix:///var/gollum.socket"
//     Acknowledge: true
//     Runlength: true
//     Sequence: true
//     Delimiter: "\n"
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
	listen      io.Closer
	protocol    string
	address     string
	delimiter   string
	flags       shared.BufferedReaderFlags
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

	if conf.GetBool("Runlength", false) {
		cons.flags |= shared.BufferedReaderFlagRLE
	}

	if conf.GetBool("Sequence", false) {
		cons.flags |= shared.BufferedReaderFlagSequence
	}

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

func clientDisconnected(err error) bool {
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

func (cons *Socket) readFromConnection(conn net.Conn) {
	defer conn.Close()
	var buffer shared.BufferedReader

	buffer = shared.NewBufferedReader(socketBufferGrowSize, cons.flags, cons.delimiter, cons.PostMessageFromSlice)

	for !cons.quit {
		err := buffer.Read(conn)

		// Handle errors
		if err != nil && err != io.EOF {
			if clientDisconnected(err) {
				return // ### return, connection closed ###
			}

			Log.Error.Print("Socket read failed: ", err)
			continue // ### continue, keep open, try again ###
		}

		// Send ack if everything was ok
		if cons.acknowledge {
			fmt.Fprint(conn, "OK")
		}
	}
}

func (cons *Socket) udpAccept() {
	cons.readFromConnection(cons.listen.(*net.UDPConn))
	cons.MarkAsDone()
}

func (cons *Socket) tcpAccept() {
	listener := cons.listen.(net.Listener)
	for !cons.quit {
		client, err := listener.Accept()
		if err != nil {
			if !cons.quit {
				Log.Error.Print("Socket listen failed: ", err)
			}
			break // ### break ###
		}

		go func() {
			defer shared.RecoverShutdown()
			cons.readFromConnection(client)
		}()
	}

	cons.MarkAsDone()
}

// Consume listens to a given socket. Messages are expected to be separated by
// either \n or \r\n.
func (cons Socket) Consume(threads *sync.WaitGroup) {
	var err error
	cons.quit = false

	if cons.protocol == "udp" {
		addr, _ := net.ResolveUDPAddr(cons.protocol, cons.address)
		if cons.listen, err = net.ListenUDP(cons.protocol, addr); err != nil {
			Log.Error.Print("Socket connection error: ", err)
			return
		}
		go func() {
			defer shared.RecoverShutdown()
			cons.udpAccept()
		}()
	} else {
		if cons.listen, err = net.Listen(cons.protocol, cons.address); err != nil {
			Log.Error.Print("Socket connection error: ", err)
			return
		}
		go func() {
			defer shared.RecoverShutdown()
			cons.tcpAccept()
		}()
	}

	defer func() {
		cons.quit = true
		cons.listen.Close()
	}()

	cons.DefaultControlLoop(threads, nil)
}
