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
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/shared"
)

// Forward formatter plugin
// Forward is a formatter that passes a message as is
// Configuration example
//
//  - "stream.Broadcast":
//    Formatter: "format.Forward"
type Forward struct {
}

func init() {
	shared.TypeRegistry.Register(Forward{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Forward) Configure(conf core.PluginConfig) error {
	return nil
}

// Format returns the original message payload
func (format *Forward) Format(msg core.Message) ([]byte, core.MessageStreamID) {
	return msg.Data, msg.StreamID
}
