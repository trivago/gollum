// Copyright 2015-2016 trivago GmbH
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
	"encoding/base64"
	"github.com/trivago/gollum/core"
)

// Serialize formatter plugin
// Serialize is a formatter that serializes a message for later retrieval.
// Configuration example
//
//  - "stream.Broadcast":
//    Formatter: "format.Serialize"
//    SerializeFormatter: "format.Envelope"
//    SerializeStringEncode: true
//
// SerializeFormatter defines the formatter for the data transferred as
// message. By default this is set to "format.Forward"
//
// SerializeStringEncode causes the serialized data to be base64 encoded and
// newline separated. This is enabled by default.
type Serialize struct {
	core.FormatterBase
	encode bool
}

func init() {
	core.TypeRegistry.Register(Serialize{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Serialize) Configure(conf core.PluginConfigReader) error {
	format.FormatterBase.Configure(conf)

	format.encode = conf.GetBool("Encode", true)
	return conf.Errors.OrNil()
}

// Format prepends the sequence number of the message (followed by ":") to the
// message.
func (format *Serialize) Format(msg *core.Message) ([]byte, core.MessageStreamID) {
	data, err := msg.Serialize()
	if err != nil {
		format.Log.Error.Print(err)
		return msg.Data, msg.StreamID
	}

	if !format.encode {
		return data, msg.StreamID // ### return, raw data ###
	}

	encodedSize := base64.StdEncoding.EncodedLen(len(data))
	encoded := make([]byte, encodedSize+1)
	base64.StdEncoding.Encode(encoded, data)
	encoded[encodedSize] = '\n'

	return encoded, msg.StreamID
}
