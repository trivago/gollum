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

// Base64Decode formatter plugin
//
// Base64Decode is a formatter that decodes base64 encoded messages.
// If a message is not or only partly base64 encoded an error will be logged
// and the decoded part is returned. RFC 4648 is expected.
//
// Parameters
//
// - Base64Dictionary: This value defines the 64-character base64 lookup
// dictionary to use. When left empty, a dictionary as defined by RFC4648 is used.
// By default this parameter is set to "".
//
// Examples
//
// This example expects base64 strings from the console and decodes them before
// transmitting the message payload.
//
//  exampleConsumer:
//    Type: consumer.Console
//    Streams: "*"
//    Modulators:
//      - format.Base64Decode
//
//
type Base64Decode struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	dictionary           *base64.Encoding
}

func init() {
	core.TypeRegistry.Register(Base64Decode{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Base64Decode) Configure(conf core.PluginConfigReader) {
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

// ApplyFormatter execute the formatter
func (format *Base64Decode) ApplyFormatter(msg *core.Message) error {
	decoded, err := format.getDecodedContent(format.GetAppliedContent(msg))
	if err != nil {
		return err
	}

	format.SetAppliedContent(msg, decoded)
	return nil
}

func (format *Base64Decode) getDecodedContent(content []byte) ([]byte, error) {
	decodedLen := format.dictionary.DecodedLen(len(content))
	decoded := make([]byte, decodedLen)

	size, err := format.dictionary.Decode(decoded, content)
	if err != nil {
		format.Logger.Error(err)
		return nil, err
	}

	return decoded[:size], nil
}
