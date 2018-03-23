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
	"io"
	"net"
	"strings"
	"sync"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo"
	"github.com/trivago/tgo/tio"
	"github.com/trivago/tgo/tnet"
)

// Proxy consumer
//
// This consumer reads messages from a given socket like consumer.Socket but
// allows reverse communication, too. Producers which require this kind of
// communication can access message.GetSource to write data back to the client
// sending the message. See producer.Proxy as an example target producer.
//
// Parameters
//
// - Address: Defines the protocol, host and port or the unix domain socket to
// listen to. This can either be any ip address and port like "localhost:5880"
// or a file like "unix:///var/gollum.socket". Only unix and tcp protocols are
// supported.
// By default this parameter is set to ":5880".
//
//
// - Partitioner: Defines the algorithm used to read messages from the router.
// The messages will be sent as a whole, no cropping or removal will take place.
// By default this parameter is set to "delimiter".
//
//  - delimiter: Separates messages by looking for a delimiter string.
//  The delimiter is removed from the message.
//
//  - ascii: Reads an ASCII number at a given offset until a given delimiter is
//  found. Everything to the left of and including the delimiter is removed
//  from the message.
//
//  - binary: reads a binary number at a given offset and size.
//  The number is removed from the message.
//
//  - binary_le: is an alias for "binary".
//
//  - binary_be: acts like "binary"_le but uses big endian encoding.
//
//  - fixed: assumes fixed size messages.
//
// - Delimiter: Defines the delimiter string used to separate messages if
// partitioner is set to "delimiter" or the string used to separate the message
// length if partitioner is set to "ascii".
// By default this parameter is set to "\n".
//
// - Offset: Defines an offset in bytes used to read the length provided for
// partitioner "binary" and "ascii".
// By default this parameter is set to 0.
//
// - Size: Defines the size of the length prefix used by partitioner "binary"
// or the message total size when using partitioner "fixed".
// When using partitioner "binary" this parameter can be set to 1,2,4 or 8 when
// using uint8,uint16,uint32 or uint64 length prefixes.
// By default this parameter is set to 4.
//
// Examples
//
// This example will accepts 64bit length encoded data on TCP port 5880.
//
//  proxyReceive:
//    Type: consumer.Proxy
//    Streams: proxyData
//    Address: ":5880"
//    Partitioner: binary
//    Size: 8
//
type Proxy struct {
	core.SimpleConsumer `gollumdoc:"embed_type"`
	listen              io.Closer
	protocol            string
	address             string
	flags               tio.BufferedReaderFlags
	delimiter           string `config:"Delimiter" default:"\n"`
	offset              int    `config:"Offset" default:"0"`
	size                int    `config:"Size" default:"4"`
}

func init() {
	core.TypeRegistry.Register(Proxy{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *Proxy) Configure(conf core.PluginConfigReader) {
	cons.protocol, cons.address = tnet.ParseAddress(conf.GetString("Address", ":5880"), "tcp")
	if cons.protocol == "udp" {
		conf.Errors.Pushf("UDP is not supported")
	}

	cons.flags = tio.BufferedReaderFlagEverything

	partitioner := strings.ToLower(conf.GetString("Partitioner", "delimiter"))
	switch partitioner {
	case "binary_be":
		cons.flags |= tio.BufferedReaderFlagBigEndian
		fallthrough

	case "binary", "binary_le":
		switch cons.size {
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
		cons.offset = cons.size

	case "ascii":
		cons.flags |= tio.BufferedReaderFlagMLE

	case "delimiter":
		// Nothing to add

	default:
		conf.Errors.Pushf("Unknown partitioner: %s", partitioner)
	}
}

func (cons *Proxy) accept() {
	defer cons.WorkerDone()

	listener := cons.listen.(net.Listener)
	for cons.IsActive() {

		client, err := listener.Accept()
		if err != nil {
			if cons.IsActive() {
				cons.Logger.Error("Listen failed: ", err)
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
		cons.Logger.Error("Connection error: ", err)
		return
	}

	go tgo.WithRecoverShutdown(func() {
		cons.AddMainWorker(workers)
		cons.accept()
	})

	defer cons.listen.Close()
	cons.ControlLoop()
}
