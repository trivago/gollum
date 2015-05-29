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
	"github.com/trivago/gollum/shared"
)

// Base64Encode is a formatter that encodes a message as base64.
// Configuration example
//
//   - "<producer|stream>":
//     Formatter: "format.Base64Encode"
type Base64Encode struct {
}

func init() {
	shared.RuntimeType.Register(Base64Encode{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Base64Encode) Configure(conf core.PluginConfig) error {
	return nil
}

// Format returns the original message payload
func (format *Base64Encode) Format(msg core.Message) ([]byte, core.MessageStreamID) {
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(msg.Data)))
	base64.StdEncoding.Encode(encoded, msg.Data)
	return encoded, msg.StreamID
}
