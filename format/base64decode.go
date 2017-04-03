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
	"encoding/base64"
	"fmt"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
)

// Base64Decode formatter plugin
// Base64Decode is a formatter that decodes a base64 message.
// If a message is not or only partly base64 encoded an error will be logged
// and the decoded part is returned. RFC 4648 is expected.
// Configuration example
//
//  - "stream.Broadcast":
//    Formatter: "format.Base64Decode"
//    Base64Formatter: "format.Forward"
//    Base64Dictionary: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz01234567890+/"
//
// Base64Dictionary defines the 64-character base64 lookup dictionary to use. When
// left empty a dictionary as defined by RFC4648 is used. This is the default.
//
// Base64DataFormatter defines a formatter that is applied before the base64
// decoding takes place. By default this is set to "format.Forward"
type Base64Decode struct {
	base       core.Formatter
	dictionary *base64.Encoding
}

func init() {
	shared.TypeRegistry.Register(Base64Decode{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Base64Decode) Configure(conf core.PluginConfig) error {
	plugin, err := core.NewPluginWithType(conf.GetString("Base64Formatter", "format.Forward"), conf)
	if err != nil {
		return err
	}
	format.base = plugin.(core.Formatter)

	dict := conf.GetString("Base64Dictionary", "")
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
func (format *Base64Decode) Format(msg core.Message) ([]byte, core.MessageStreamID) {
	data, streamID := format.base.Format(msg)
	decoded := make([]byte, format.dictionary.DecodedLen(len(data)))
	size, err := format.dictionary.Decode(decoded, data)
	if err != nil {
		Log.Error.Print("Base64Decode: ", err)
		return data, streamID
	}
	return decoded[:size], streamID
}
