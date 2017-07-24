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
	"github.com/trivago/tgo/tnet"
	"gopkg.in/mcuadros/go-syslog.v2"
	"gopkg.in/mcuadros/go-syslog.v2/format"
	"sync"
)

// Syslogd consumer plugin
//
// The syslogd consumer creates a syslogd-compatible log server and
// receives messages on a TCP or UDP port or a UNIX filesystem socket.
//
// Parameters
//
// - Address: Defines the IP address or UNIX socket to listen to.
// This can take one of four forms, to listen on a TCP, UDP or UNIX domain
// socket:
//
// * [hostname|ip]:<tcp-port>
// * tcp://<hostname|ip>:<tcp-port>
// * udp://<hostname|ip>:<udp-port>
// * unix://<filesystem-path>
//
// However, see the "Format" option for details on transport support by different
// formats. Default: "udp://0.0.0.0:514"
//
// - Format: Defines which syslog the server will support. Three standards are
// currently available:
//
// * RFC3164 (https://tools.ietf.org/html/rfc3164) - unix, udp
// * RFC5424 (https://tools.ietf.org/html/rfc5424) - unix, udp
// * RFC6587 (https://tools.ietf.org/html/rfc6587) - unix, upd, tcp
//
// All of the formats support listening to UDP and UNIX domain sockets. RFC6587
// additionally supports TCP sockets. Default: "RFC6587".
//
// Example
//
//  # Replace the system's standard syslogd with Gollum
//  "SyslogdSocketConsumer":
//    Streams: "system_syslog"
//    Address: "unix:///dev/log"
//    Format: "RFC3164"
//
//  # Listen on a TCP socket
//  "SyslogdTCPSocketConsumer":
//    Streams: "tcp_syslog"
//    Address: "tcp://0.0.0.0:5599"
//    Format: "RFC6587"
//
type Syslogd struct {
	core.SimpleConsumer `gollumdoc:"embed_type"`
	format              format.Format // RFC3164, RFC5424 or RFC6587?
	protocol            string
	address             string
}

func init() {
	core.TypeRegistry.Register(Syslogd{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *Syslogd) Configure(conf core.PluginConfigReader) {
	cons.protocol, cons.address = tnet.ParseAddress(
		conf.GetString("Address", "udp://0.0.0.0:514"), "tcp")

	format := conf.GetString("Format", "RFC6587")

	switch cons.protocol {
	case "udp", "tcp", "unix":
	default:
		conf.Errors.Pushf("Unknown protocol type %s", cons.protocol) // ### return, unknown protocol ###
	}

	switch format {
	// http://www.ietf.org/rfc/rfc3164.txt
	case "RFC3164":
		cons.format = syslog.RFC3164
		if cons.protocol == "tcp" {
			cons.Logger.Warning("RFC3164 demands UDP")
			cons.protocol = "udp"
		}

	// https://tools.ietf.org/html/rfc5424
	case "RFC5424":
		cons.format = syslog.RFC5424
		if cons.protocol == "tcp" {
			cons.Logger.Warning("RFC5424 demands UDP")
			cons.protocol = "udp"
		}

	// https://tools.ietf.org/html/rfc6587
	case "RFC6587":
		cons.format = syslog.RFC6587

	default:
		conf.Errors.Pushf("Format %s is not supported", format)
	}
}

// Handle implements the syslog handle interface
func (cons *Syslogd) Handle(parts format.LogParts, code int64, err error) {
	content := ""
	isString := false

	switch cons.format {
	case syslog.RFC3164:
		content, isString = parts["content"].(string)
	case syslog.RFC5424, syslog.RFC6587:
		content, isString = parts["message"].(string)
	default:
		cons.Logger.Error("Could not determine the format to retrieve message/content")
	}

	if !isString {
		cons.Logger.Error("Message/Content is not a string")
		return
	}

	cons.Enqueue([]byte(content))
}

// Consume opens a new syslog socket.
// Messages are expected to be separated by \n.
func (cons *Syslogd) Consume(workers *sync.WaitGroup) {
	server := syslog.NewServer()
	server.SetFormat(cons.format)
	server.SetHandler(cons)

	switch cons.protocol {
	case "unix":
		if err := server.ListenUnixgram(cons.address); err != nil {
			cons.Logger.Error("Failed to open unix://", cons.address)
		}
	case "udp":
		if err := server.ListenUDP(cons.address); err != nil {
			cons.Logger.Error("Failed to open udp://", cons.address)
		}
	case "tcp":
		if err := server.ListenTCP(cons.address); err != nil {
			cons.Logger.Error("Failed to open tcp://", cons.address)
		}
	}

	server.Boot()
	defer server.Kill()

	cons.ControlLoop()
	server.Wait()
}
