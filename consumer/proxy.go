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
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo"
	"github.com/trivago/tgo/tio"
	"github.com/trivago/tgo/tnet"
	"github.com/trivago/tgo/tstrings"
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
//
// The proxy consumer reads messages directly as-is from a given socket.
// Messages are extracted by standard message size algorithms (see Partitioner).
// This consumer can be used with any compatible proxy producer to establish
// a two-way communication.
//
// Configuration example
//
//  - "consumer.Proxy":
//    Address: ":5880"
//    Partitioner: "delimiter"
//    Delimiter: "\n"
//    Offset: 0
//    Size: 1
//
// Address defines the protocol, host and port or socket to bind to.
// This can either be any ip address and port like "localhost:5880" or a file
// like "unix:///var/gollum.socket". By default this is set to ":5880".
// UDP is not supported.
//
// Partitioner defines the algorithm used to read messages from the router.
// The messages will be sent as a whole, no cropping or removal will take place.
// By default this is set to "delimiter".
//  * "delimiter" separates messages by looking for a delimiter string.
//    The delimiter is included into the left hand message.
//  * "ascii" reads an ASCII number at a given offset until a given delimiter is found.
//    Everything to the right of and including the delimiter is removed from the message.
//  * "binary" reads a binary number at a given offset and size.
//  * "binary_le" is an alias for "binary".
//  * "binary_be" is the same as "binary" but uses big endian encoding.
//  * "fixed" assumes fixed size messages.
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
type Proxy struct {
	core.SimpleConsumer
	listen    io.Closer
	protocol  string
	address   string
	flags     tio.BufferedReaderFlags
	delimiter string
	offset    int
}

func init() {
	core.TypeRegistry.Register(Proxy{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *Proxy) Configure(conf core.PluginConfigReader) error {
	cons.SimpleConsumer.Configure(conf)

	cons.protocol, cons.address = tnet.ParseAddress(conf.GetString("Address", ":5880"), "tcp")
	if cons.protocol == "udp" {
		conf.Errors.Pushf("UDP is not supported")
	}

	cons.delimiter = tstrings.Unescape(conf.GetString("Delimiter", "\n"))
	cons.offset = conf.GetInt("Offset", 0)
	cons.flags = tio.BufferedReaderFlagEverything

	partitioner := strings.ToLower(conf.GetString("Partitioner", "delimiter"))
	switch partitioner {
	case "binary_be":
		cons.flags |= tio.BufferedReaderFlagBigEndian
		fallthrough

	case "binary", "binary_le":
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
		cons.offset = conf.GetInt("Size", 1)

	case "ascii":
		cons.flags |= tio.BufferedReaderFlagMLE

	case "delimiter":
		// Nothing to add

	default:
		conf.Errors.Pushf("Unknown partitioner: %s", partitioner)
	}

	return conf.Errors.OrNil()
}

func (cons *Proxy) accept() {
	defer cons.WorkerDone()

	listener := cons.listen.(net.Listener)
	for cons.IsActive() {

		client, err := listener.Accept()
		if err != nil {
			if cons.IsActive() {
				cons.Log.Error.Print("Listen failed: ", err)
			}
			break // ### break ###
		}

		go listenToProxyClient(client, cons)
	}
}

// Consume listens to a given socket.
func (cons *Proxy) Consume(workers *sync.WaitGroup) {
	var err error

	if cons.listen, err = net.Listen(cons.protocol, cons.address); err != nil {
		cons.Log.Error.Print("Connection error: ", err)
		return
	}

	go tgo.WithRecoverShutdown(func() {
		cons.AddMainWorker(workers)
		cons.accept()
	})

	defer cons.listen.Close()
	cons.ControlLoop()
}
