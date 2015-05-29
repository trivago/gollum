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
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
)

// Base64Decode is a formatter that decodes a base64 message.
// If a message is not or only partly base64 encoded an error will be logged
// and the decoded part is returned. RFC 4648 is expected.
// Configuration example
//
//   - "<producer|stream>":
//     Formatter: "format.Base64Decode"
type Base64Decode struct {
}

func init() {
	shared.RuntimeType.Register(Base64Decode{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Base64Decode) Configure(conf core.PluginConfig) error {
	return nil
}

// Format returns the original message payload
func (format *Base64Decode) Format(msg core.Message) ([]byte, core.MessageStreamID) {
	decoded := make([]byte, base64.StdEncoding.DecodedLen(len(msg.Data)))
	size, err := base64.StdEncoding.Decode(decoded, msg.Data)
	if err != nil {
		Log.Error.Print("Base64Decode: ", err)
	}
	return decoded[:size], msg.StreamID
}
