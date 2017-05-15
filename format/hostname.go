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

package format

import (
	"github.com/trivago/gollum/core"
	"os"
)

// Hostname formatter plugin
// Hostname is a formatter that prefixes a message with the hostname.
// Configuration example
//
//  - "stream.Broadcast":
//    Formatter: "format.Hostname"
//    HostnameFormatter: "format.Envelope"
//    HostnameSeparator: " "
//
// HostnameDataFormatter defines the formatter for the data transferred as
// message. By default this is set to "format.Envelope"
//
// HostnameSeparator sets the separator character placed after the hostname.
// This is set to " " by default.
type Hostname struct {
	core.SimpleFormatter `gollumdoc:embed_type`
	separator []byte
}

func init() {
	core.TypeRegistry.Register(Hostname{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Hostname) Configure(conf core.PluginConfigReader) error {
	format.SimpleFormatter.Configure(conf)

	format.separator = []byte(conf.GetString("Separator", ":"))
	return conf.Errors.OrNil()
}

// ApplyFormatter update message payload
func (format *Hostname) ApplyFormatter(msg *core.Message) error {
	hostname, err := os.Hostname()
	if err != nil {
		format.Log.Error.Print(err)
		hostname = "unknown host"
	}

	dataSize := len(hostname) + len(format.separator) + msg.Len()
	payload := core.MessageDataPool.Get(dataSize)

	offset := copy(payload, []byte(hostname))
	offset += copy(payload[offset:], format.separator)
	copy(payload[offset:], msg.Data())

	msg.Store(payload)
	return nil
}
