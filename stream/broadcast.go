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
// Stream defines the streams this stream plugin binds to (i.e. the streams
// affected by this config).
//
// Formatter defines a formatter that is applied to all messages sent to this
// stream. This can be used to bring different streams to the same format
// required by a producer formatter. By default this is set to format.Forward.
//
// Filter defines a filter function that removes or allows certain messages to
// pass through this stream. By default this is set to filter.All.
type Broadcast struct {
	core.StreamBase
}

func init() {
	shared.RuntimeType.Register(Broadcast{})
}
