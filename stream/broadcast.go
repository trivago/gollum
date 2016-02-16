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

package stream

import (
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/shared"
)

// Broadcast stream plugin
// Messages will be sent to all producers attached to this stream.
type Broadcast struct {
	core.StreamBase
}

func init() {
	shared.TypeRegistry.Register(Broadcast{})
}

// Configure initializes this distributor with values from a plugin config.
func (stream *Broadcast) Configure(conf core.PluginConfig) error {
	return stream.StreamBase.ConfigureStream(conf, stream.Broadcast)
}
