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
	"github.com/jeromer/syslogparser"
	"github.com/mcuadros/go-syslog"
	"github.com/mcuadros/go-syslog/format"
	"github.com/trivago/gollum/shared"
	"sync"
)

// Syslogd consumer plugin
// Configuration example
//
//   - "consumer.Syslogd":
//     Enable: true
//     Address: "0.0.0.0:5880"
//     Format: "RFC3164"
//     Protocol: "udp"
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
	sequence *uint64
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
	cons.sequence = new(uint64)

	// TODO Support unix domain socket (see Socket consumer)

	return err
}

// Handle implements the syslog handle interface
func (cons Syslogd) Handle(parts syslogparser.LogParts, code int64, err error) {
	content, isString := parts["content"].(string)
	if isString {
		cons.PostMessageString(content, *cons.sequence)
		*cons.sequence++
	}
}

// Consume opens a new syslog socket.
// Messages are expected to be separated by \n.
func (cons Syslogd) Consume(threads *sync.WaitGroup) {
	server := syslog.NewServer()
	server.SetFormat(cons.format)
	server.SetHandler(cons)

	switch cons.protocol {
	case "udp":
		server.ListenUDP(cons.address)
	case "tcp":
		server.ListenTCP(cons.address)
	}

	server.Boot()
	defer func() {
		server.Kill()
		cons.MarkAsDone()
	}()

	cons.DefaultControlLoop(threads, nil)
}
