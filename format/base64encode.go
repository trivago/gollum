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

// Base64Encode formatter plugin
// Base64Encode is a formatter that encodes a message as base64.
// Configuration example
//
//  - "stream.Broadcast":
//    Formatter: "format.Base64Encode"
//    Base64Formatter: "format.Forward"
//    Base64Dictionary: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz01234567890+/"
//
// Base64Dictionary defines the 64-character base64 lookup dictionary to use.
// When left empty a dictionary as defined by RFC4648 is used. This is the default.
//
// Base64DataFormatter defines a formatter that is applied before the base64
// encoding takes place. By default this is set to "format.Forward"
type Base64Encode struct {
	core.SimpleFormatter
	dictionary *base64.Encoding
}

func init() {
	core.TypeRegistry.Register(Base64Encode{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Base64Encode) Configure(conf core.PluginConfigReader) error {
	format.SimpleFormatter.Configure(conf)

	dict := conf.GetString("Dictionary", "")
	if dict == "" {
		format.dictionary = base64.StdEncoding
	} else {
		if len(dict) != 64 {
			conf.Errors.Pushf("Base64 dictionary must contain 64 characters.")
		}
		format.dictionary = base64.NewEncoding(dict)
	}
	return conf.Errors.OrNil()
}

// Format returns the original message payload
func (format *Base64Encode) Format(msg *core.Message) {
	encodedLen := format.dictionary.EncodedLen(msg.Len())
	encoded := core.MessageDataPool.Get(encodedLen)

	format.dictionary.Encode(encoded, msg.Data())
	msg.Store(encoded)
}
