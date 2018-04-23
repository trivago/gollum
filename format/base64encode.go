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

package format

import (
	"encoding/base64"

	"github.com/trivago/gollum/core"
)

// Base64Encode formatter
//
// Base64Encode is a formatter that decodes Base64 encoded strings. Custom dictionaries
// are supported, by default RFC 4648 standard encoding is used.
//
// Parameters
//
// - Base64Dictionary: Defines the 64-character base64 lookup dictionary to use.
// When left empty a RFC 4648 standard encoding is used.
// By default this parameter is set to "".
//
// Examples
//
// This example uses RFC 4648 URL encoding to format incoming data.
//
//  ExampleConsumer:
//    Type: consumer.Console
//    Streams: console
//    Modulators:
//      - formatter.Base64Encode
//        Dictionary: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
//
type Base64Encode struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	dictionary           *base64.Encoding
}

func init() {
	core.TypeRegistry.Register(Base64Encode{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Base64Encode) Configure(conf core.PluginConfigReader) {
	dict := conf.GetString("Dictionary", "")
	if dict == "" {
		format.dictionary = base64.StdEncoding
	} else {
		if len(dict) != 64 {
			conf.Errors.Pushf("Base64 dictionary must contain 64 characters.")
		}
		format.dictionary = base64.NewEncoding(dict)
	}
}

// ApplyFormatter update message payload
func (format *Base64Encode) ApplyFormatter(msg *core.Message) error {
	encoded := format.getEncodedContent(format.GetAppliedContent(msg))
	format.SetAppliedContent(msg, encoded)

	return nil
}

func (format *Base64Encode) getEncodedContent(content []byte) []byte {
	encodedLen := format.dictionary.EncodedLen(len(content))
	encoded := make([]byte, encodedLen)

	format.dictionary.Encode(encoded, content)
	return encoded
}
