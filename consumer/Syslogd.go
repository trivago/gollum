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
	"github.com/trivago/gollum/shared"
	"gopkg.in/mcuadros/go-syslog.v2"
	"gopkg.in/mcuadros/go-syslog.v2/format"
	"sync"
)

// Syslogd consumer plugin
// Configuration example
//
//   - "consumer.Syslogd":
//     Enable: true
//     Address: 0.0.0.0:5880
//     Format: RFC3164
//     Protocol: udp
//
// Address stores the identifier to bind to.
// This must be a ip address and port like "127.0.0.1:5880"
// Unix domain sockets are not supported yet.
// By default this is set to "0.0.0.0:5880".
//
// Format define the used syslog standard.
// Three standards are currently supported:
// 	* RFC3164 (https://tools.ietf.org/html/rfc3164)
// 	* RFC5424 (https://tools.ietf.org/html/rfc5424)
// 	* RFC6587 (https://tools.ietf.org/html/rfc6587)
// This format can make choices for the used protocol or message parts.
// Please have a look at the application which will produce syslog messages
// which format is supported.
// By default this is set to "RFC5424".
//
// Protocol specifies the transport layer of syslog messages.
// Supported protocols are:
//	* udp
//	* tcp
// Depending on the chosen format (see above) a protocol will be set by Gollum,
// because some RFCs describe their protocol:
// 	* RFC3164: udp
// 	* RFC6587: tcp
// For format "RFC5424" the protocol can be chosen.
// By default this is set to "udp".
type Syslogd struct {
	shared.ConsumerBase
	format   format.Format // RFC3164, RFC5424 or RFC6587?
	protocol string        // udp or tcp
	address  string
}

func init() {
	shared.RuntimeType.Register(Syslogd{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *Syslogd) Configure(conf shared.PluginConfig) error {
	err := cons.ConsumerBase.Configure(conf)
	if err != nil {
		return err
	}

	format := conf.GetString("Format", "RFC5424")
	switch format {
	// http://www.ietf.org/rfc/rfc3164.txt
	case "RFC3164":
		cons.format = syslog.RFC3164
		cons.protocol = "udp"

	// https://tools.ietf.org/html/rfc5424
	case "RFC5424":
		cons.format = syslog.RFC5424
		cons.protocol = conf.GetString("Protocol", "udp")

	// https://tools.ietf.org/html/rfc6587
	case "RFC6587":
		cons.format = syslog.RFC6587
		cons.protocol = "tcp"
	default:
		err = fmt.Errorf("Syslog: Format %s is not supported", format)
	}

	// Port 514 is the standard port of syslog
	// Port 5880 is the port mentioned by go-syslog package
	cons.address = conf.GetString("Address", "0.0.0.0:5880")

	// TODO Support unix domain socket (see Socket consumer)

	return err
}

func (cons *Syslogd) recieve(channel syslog.LogPartsChannel) {
	defer cons.MarkAsDone()
	seq := uint64(0)

	for {
		logParts, ok := <-channel
		if !ok || logParts == nil {
			return
		}

		// TODO The complete message / map can be marshalled as json?
		// TODO Maybe some message formatter can help?
		// fmt.Println(logParts["content"])
		fmt.Println(logParts)

		cons.PostMessage((logParts["content"]).(string), seq)
		seq++
	}
}

// Consume opens a new syslog socket.
// Messages are expected to be separated by \n.
func (cons Syslogd) Consume(threads *sync.WaitGroup) {

	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)

	server := syslog.NewServer()
	server.SetFormat(cons.format)
	server.SetHandler(handler)

	switch cons.protocol {
	case "udp":
		server.ListenUDP(cons.address)
	case "tcp":
		server.ListenTCP(cons.address)
	}

	server.Boot()
	defer func() {
		server.Wait()
		close(channel)
	}()

	go cons.recieve(channel)
	cons.DefaultControlLoop(threads, nil)
}
