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

package stream

import (
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/shared"
)

// Broadcast stream plugin
// Configuration example
//
//   - "stream.Broadcast":
//     Enable: true
//     Stream: "data"
//	   Formatter: "format.Envelope"
//     Filter: "filter.All"
//
// Messages will be sent to all producers attached to this stream.
//
// Stream defines the stream this stream plugin binds to.
//
// Formatter defines a formatter that is applied to all messages sent to this
// stream. This can be used to bring different streams to the same format
// required by a producer formatter. By default this is set to format.Forward.
//
// Filter defines a filter function that removes or allows certain messages to
// pass through this stream. By default this is set to filter.All.
//
// TimeoutMs sets a timeout in milliseconds for messages to wait if a producer's
// queue is full. This will actually overwrite the ChannelTimeoutMs value for
// any attached producer and follows the same rules.
// If no value is set, the producer's timout value is used.
type Broadcast struct {
	core.StreamBase
}

func init() {
	shared.RuntimeType.Register(Broadcast{})
}

// Configure initializes this distributor with values from a plugin config.
func (stream *Broadcast) Configure(conf core.PluginConfig) error {
	return stream.StreamBase.ConfigureStream(conf, stream.Broadcast)
}
