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
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	"io"
	"net"
	"strings"
	"sync"
)

type proxyPartitioner int

const (
	proxyPartDelimiter = proxyPartitioner(iota)
	proxyPartBinary    = proxyPartitioner(iota)
	proxyPartText      = proxyPartitioner(iota)
)

// Proxy consumer plugin.
// This plugin enables two-way communication when sending to a proxy producer.
// Configuration example
//
//   - "consumer.Proxy":
//     Enable: true
//     Address: "unix:///var/gollum.socket"
//
// The socket consumer reads messages directly as-is from a given socket.
// It does support a minimal protocol for sending messagelength and sequence
// number.
//
// Address stores the identifier to bind to.
// This can either be any ip address and port like "localhost:5880" or a file
// like "unix:///var/gollum.socket". By default this is set to ":5880".
// UDP is not supported.
//
// ClientBuffer defines the number of message responses that may be buffered for
// each client. By default this is set to 32.
//
// Partitioner defines the algorithm used to read messages from the stream.
// The messages will be sent as a whole, no cropping or removal will take place.
//  - "delimiter" separates messages by looking for a delimiter string. The
//    delimiter is included into the left hand message.
//  - "binary" reads a binary number at a given offset and size
//  - "binary_le" is an alias for "binary"
//  - "binary_be" is the same as "binary" but uses big endian encoding
//  - "fixed" assumes fixed size messages
//  - "text" reads an ASCII encoded number at a given offset until a given
//    delimiter is found.
// By default this is set to "delimiter".
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
type Proxy struct {
	core.ConsumerBase
	listen       io.Closer
	protocol     string
	address      string
	flags        shared.BufferedReaderFlags
	delimiter    string
	offset       int
	clientBuffer int
	quit         bool
}

func init() {
	shared.RuntimeType.Register(Proxy{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *Proxy) Configure(conf core.PluginConfig) error {
	err := cons.ConsumerBase.Configure(conf)
	if err != nil {
		return err
	}

	cons.address, cons.protocol = shared.ParseAddress(conf.GetString("Address", ":5880"))
	if cons.protocol == "udp" {
		return fmt.Errorf("Proxy does not support UDP")
	}

	escapeChars := strings.NewReplacer("\\n", "\n", "\\r", "\r", "\\t", "\t")
	cons.delimiter = escapeChars.Replace(conf.GetString("Delimiter", "\n"))
	cons.offset = conf.GetInt("Offset", 0)
	cons.clientBuffer = conf.GetInt("ClientBuffer", 32)
	cons.flags = shared.BufferedReaderFlagEverything

	partitioner := strings.ToLower(conf.GetString("Partitioner", "delimiter"))
	switch partitioner {
	case "binary_be":
		cons.flags |= shared.BufferedReaderFlagBigEndian
		fallthrough

	case "binary", "binary_le":
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

	case "text":
		cons.flags |= shared.BufferedReaderFlagMLE

	case "delimiter":
		// Nothing to add

	default:
		return fmt.Errorf("Unknown partitioner: %s", partitioner)
	}

	cons.quit = false
	return err
}

func (cons *Proxy) accept() {
	defer cons.WorkerDone()

	listener := cons.listen.(net.Listener)
	for !cons.quit {
		client, err := listener.Accept()
		if err != nil {
			if !cons.quit {
				Log.Error.Print("Proxy listen failed: ", err)
			}
			break // ### break ###
		}

		go listenToProxyClient(client, cons)
	}
}

// Consume listens to a given socket.
func (cons *Proxy) Consume(workers *sync.WaitGroup) {
	var err error
	cons.quit = false

	if cons.listen, err = net.Listen(cons.protocol, cons.address); err != nil {
		Log.Error.Print("Proxy connection error: ", err)
		return
	}

	go func() {
		defer shared.RecoverShutdown()
		cons.AddMainWorker(workers)
		cons.accept()
	}()

	defer func() {
		cons.quit = true
		cons.listen.Close()
	}()

	cons.DefaultControlLoop(nil)
}
