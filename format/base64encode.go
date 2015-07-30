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

package format

import (
	"encoding/base64"
	"fmt"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/shared"
)

// Base64Encode is a formatter that encodes a message as base64.
// Configuration example
//
//   - "<producer|stream>":
//     Formatter: "format.Base64Encode"
//     Base64Formatter: "format.Forward"
//     Base64Dictionary: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz01234567890+/"
//
// Base64Dictionary defines the 64-character base64 lookup dictionary to use.
// When left empty a dictionary as defined by RFC4648 is used. This is the default.
//
// Base64DataFormatter defines a formatter that is applied before the base64
// encoding takes place. By default this is set to "format.Forward"
type Base64Encode struct {
	base       core.Formatter
	dictionary *base64.Encoding
}

func init() {
	shared.TypeRegistry.Register(Base64Encode{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Base64Encode) Configure(conf core.PluginConfig) error {
	plugin, err := core.NewPluginWithType(conf.GetString("Base64DataFormatter", "format.Forward"), conf)
	if err != nil {
		return err
	}
	format.base = plugin.(core.Formatter)

	dict := conf.GetString("Base64Formatter", "")
	if dict == "" {
		format.dictionary = base64.StdEncoding
	} else {
		if len(dict) != 64 {
			return fmt.Errorf("Base64 dictionary must contain 64 characters.")
		}
		format.dictionary = base64.NewEncoding(dict)
	}
	return nil
}

// Format returns the original message payload
func (format *Base64Encode) Format(msg core.Message) ([]byte, core.MessageStreamID) {
	data, streamID := format.base.Format(msg)
	encoded := make([]byte, format.dictionary.EncodedLen(len(data)))
	format.dictionary.Encode(encoded, data)
	return encoded, streamID
}
