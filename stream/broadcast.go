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
)

// Broadcast stream plugin
// Messages will be sent to all producers attached to this stream.
type Broadcast struct {
	core.SimpleStream
}

func init() {
	core.TypeRegistry.Register(Broadcast{})
}

// Configure initializes this distributor with values from a plugin config.
func (stream *Broadcast) Configure(conf core.PluginConfigReader) error {
	return stream.SimpleStream.Configure(conf)
}

func (stream *Broadcast) Enqueue(msg *core.Message) bool {
	producers := stream.GetProducers()
	if len(producers) == 0 {
		return false // ### return, no route to producer ###
	}

	if len(producers) == 1 {
		producers[0].Enqueue(msg, stream.Timeout)
		return true // ### return, fast path ###
	}

	for _, prod := range stream.GetProducers() {
		msg := msg.Clone()
		prod.Enqueue(msg, stream.Timeout)
	}

	return true
}
