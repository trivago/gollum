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
	"os"
	"strings"
	"sync"
	"time"

	"github.com/trivago/tgo/tcontainer"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tnet"
	syslog "gopkg.in/mcuadros/go-syslog.v2"
	"gopkg.in/mcuadros/go-syslog.v2/format"
)

// Syslogd consumer plugin
//
// The syslogd consumer creates a syslogd-compatible log server and
// receives messages on a TCP or UDP port or a UNIX filesystem socket.
//
// Parameters
//
// - Address: Defines the IP address or UNIX socket to listen to.
// This can take one of the four forms below, to listen on a TCP, UDP
// or UNIX domain socket. However, see the "Format" option for details on
// transport support by different formats.
// * [hostname|ip]:<tcp-port>
// * tcp://<hostname|ip>:<tcp-port>
// * udp://<hostname|ip>:<udp-port>
// * unix://<filesystem-path>
// By default this parameter is set to "udp://0.0.0.0:514"
//
// - Format: Defines which syslog standard the server will support.
// Three standards, listed below, are currently available.  All
// standards support listening to UDP and UNIX domain sockets.
// RFC6587 additionally supports TCP sockets. Default: "RFC6587".
// * RFC3164 (https://tools.ietf.org/html/rfc3164) - unix, udp
// * RFC5424 (https://tools.ietf.org/html/rfc5424) - unix, udp
// * RFC6587 (https://tools.ietf.org/html/rfc6587) - unix, upd, tcp
// By default this parameter is set to "RFC6587".
//
// - Permissions: This value sets the filesystem permissions
// as a four-digit octal number in case the address is a Unix domain socket
// (i.e. unix://<filesystem-path>).
// By default this parameter is set to "0770".
//
// - SetMetadata: When set to true, syslog based metadata will be attached to
// the message. The metadata fields added depend on the protocol version used.
// RFC3164 supports: tag, timestamp, hostname, priority, facility, severity.
// RFC5424 and RFC6587 support: app_name, version, proc_id , msg_id, timestamp,
// hostname, priority, facility, severity.
// By default this parameter is set to "false".
//
// - TimestampFormat: When using SetMetadata this string denotes the go time
// format used to convert syslog timestamps into strings.
// By default this parameter is set to "2006-01-02T15:04:05.000 MST".
//
// Examples
//
// Replace the system's standard syslogd with Gollum
//
//  SyslogdSocketConsumer:
//    Type: consumer.Syslogd
//    Streams: "system_syslog"
//    Address: "unix:///dev/log"
//    Format: "RFC3164"
//
// Listen on a TCP socket
//
//  SyslogdTCPSocketConsumer:
//    Type: consumer.Syslogd
//    Streams: "tcp_syslog"
//    Address: "tcp://0.0.0.0:5599"
//    Format: "RFC6587"
//
type Syslogd struct {
	core.SimpleConsumer `gollumdoc:"embed_type"`
	format              format.Format // RFC3164, RFC5424 or RFC6587?
	protocol            string
	address             string
	withMetadata        bool        `config:"SetMetadata" default:"false"`
	fileFlags           os.FileMode `config:"Permissions" default:"0770"`
	timestampFormat     string      `config:"TimestampFormat" default:"2006-01-02T15:04:05.000 MST"`
}

func init() {
	core.TypeRegistry.Register(Syslogd{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *Syslogd) Configure(conf core.PluginConfigReader) {
	cons.protocol, cons.address = tnet.ParseAddress(
		conf.GetString("Address", "udp://0.0.0.0:514"), "tcp")

	syslogFormat := conf.GetString("Format", "RFC6587")

	switch cons.protocol {
	case "udp", "tcp", "unix":
	default:
		conf.Errors.Pushf("Unknown protocol type %s", cons.protocol) // ### return, unknown protocol ###
	}

	switch syslogFormat {
	// http://www.ietf.org/rfc/rfc3164.txt
	case "RFC3164":
		cons.format = syslog.RFC3164
		if cons.protocol != "udp" {
			cons.Logger.Infof("Using '%s' instead of 'udp' for RFC3164 violates the standard", cons.protocol)
		}

	// https://tools.ietf.org/html/rfc5424
	case "RFC5424":
		cons.format = syslog.RFC5424
		if cons.protocol != "udp" {
			cons.Logger.Infof("Using '%s' instead of 'udp' for RFC5424 violates the standard", cons.protocol)
		}

	// https://tools.ietf.org/html/rfc6587
	case "RFC6587":
		cons.format = syslog.RFC6587

	default:
		conf.Errors.Pushf("Format %s is not supported", syslogFormat)
	}
}

func parseCustomFields(data string, metadata *tcontainer.MarshalMap) {
	if len(data) == 0 {
		return
	}

	endOfSid := strings.IndexByte(data, ' ')
	if endOfSid < 0 {
		return
	}

	data = data[endOfSid+1:]
	for {
		if len(data) == 0 {
			return // ### return, eof ###
		}
		endOfKey := strings.IndexByte(data, '=')
		if endOfKey < 0 {
			return // ### return, end of data ###
		}

		key := strings.TrimSpace(data[:endOfKey])
		// we might cross into a new section while searching for a key
		if sectionStart := strings.IndexByte(key, '['); sectionStart >= 0 {
			data = data[sectionStart+1:]
			endOfSid = strings.IndexByte(data, ' ')
			if endOfSid < 0 {
				return // ### return, end of data ###
			}
			data = data[endOfSid+1:]
			continue // ### continue, look for new key ###
		}

		data = data[endOfKey+1:]

		startOfValue := strings.IndexByte(data, '"') + 1
		if startOfValue < 1 {
			return // ### return, end of data ###
		}

		i := startOfValue
		var endOfValue int
		hasQuotes := false
		for {
			endOfValue = strings.IndexByte(data[i:], '"')
			if endOfValue < 0 {
				return // ### return, end of data ###
			}
			endOfValue += i
			if data[endOfValue-1] == '\\' {
				hasQuotes = true
				i = endOfValue + 1
				continue // ### continue, escaped quote ###
			}
			break // ### break, done ###
		}

		value := data[startOfValue:endOfValue]
		if hasQuotes {
			value = strings.Replace(value, "\\\"", "\"", -1)
		}

		metadata.Set(key, value)
		data = data[endOfValue+1:]
	}
}

// Handle implements the syslog handle interface
func (cons *Syslogd) Handle(parts format.LogParts, code int64, err error) {
	content := ""
	isString := false
	metaData := core.NewMetadata()

	switch cons.format {
	case syslog.RFC3164:
		content, isString = parts["content"].(string)

		if cons.withMetadata {
			hostname, _ := parts["hostname"].(string)
			tag, _ := parts["tag"].(string)
			priority, _ := parts["priority"].(int)
			facility, _ := parts["facility"].(int)
			severity, _ := parts["severity"].(int)
			timestamp, _ := parts["timestamp"].(time.Time)

			metaData.Set("tag", tag)
			metaData.Set("timestamp", timestamp.Format(cons.timestampFormat))

			metaData.Set("hostname", hostname)
			metaData.Set("priority", priority)
			metaData.Set("facility", facility)
			metaData.Set("severity", severity)
		}

	case syslog.RFC5424, syslog.RFC6587:
		content, isString = parts["message"].(string)

		if cons.withMetadata {
			hostname, _ := parts["hostname"].(string)
			app, _ := parts["app_name"].(string)
			version, _ := parts["version"].(string)
			procID, _ := parts["proc_id"].(string)
			msgID, _ := parts["msg_id"].(string)
			priority, _ := parts["priority"].(int)
			facility, _ := parts["facility"].(int)
			severity, _ := parts["severity"].(int)
			timestamp, _ := parts["timestamp"].(time.Time)
			structuredData, _ := parts["structured_data"].(string)

			metaData.Set("structured_data", structuredData)

			parseCustomFields(structuredData, &metaData)

			metaData.Set("app_name", app)
			metaData.Set("version", version)
			metaData.Set("proc_id", procID)
			metaData.Set("msg_id", msgID)
			metaData.Set("timestamp", timestamp.Format(cons.timestampFormat))

			metaData.Set("hostname", hostname)
			metaData.Set("priority", priority)
			metaData.Set("facility", facility)
			metaData.Set("severity", severity)
		}

	default:
		cons.Logger.Error("Could not determine the format to retrieve message/content")
	}

	if !isString {
		cons.Logger.Error("Message/Content is not a string")
		return
	}

	if cons.withMetadata {
		cons.EnqueueWithMetadata([]byte(content), metaData)
	} else {
		cons.Enqueue([]byte(content))
	}
}

// Consume opens a new syslog socket.
// Messages are expected to be separated by \n.
func (cons *Syslogd) Consume(workers *sync.WaitGroup) {
	server := syslog.NewServer()
	server.SetFormat(cons.format)
	server.SetHandler(cons)

	switch cons.protocol {
	case "unix":
		err := os.Chmod(cons.address, cons.fileFlags)
		if err != nil {
			cons.Logger.WithError(err).Error("Failed to set file permissions on", cons.address)
		}

		if err := server.ListenUnixgram(cons.address); err != nil {
			if errRemove := os.Remove(cons.address); errRemove != nil {
				cons.Logger.WithError(errRemove).Error("Failed to remove existing socket")
			} else {
				cons.Logger.Warning("Found existing socket ", cons.address, ". Removing.")
				err = server.ListenUnixgram(cons.address)
			}

			if err != nil {
				cons.Logger.WithError(err).Error("Failed to open unix://", cons.address)
			}
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
