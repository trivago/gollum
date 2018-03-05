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

package router

import (
	"github.com/trivago/gollum/core"
)

// Metadata router
//
// This router routes the message to a stream given in a specified metadata
// field. If the field is not set, the message will be passed along.
//
// Parameters
//
// - Key: The metadata field to read from.
// By default this parameter is set to "Stream"
//
// Examples
//
//  switchRoute:
//    Type: router.Metadata
//    Stream: errorlogs
//    Key: key
type Metadata struct {
	Broadcast `gollumdoc:"embed_type"`
	key       string `config:"Key" default:"Stream"`
}

func init() {
	core.TypeRegistry.Register(Metadata{})
}

// Configure initializes this distributor with values from a plugin config.
func (router *Metadata) Configure(conf core.PluginConfigReader) {
}

// Start the router
func (router *Metadata) Start() error {
	return nil
}

// Enqueue enques a message to the router
func (router *Metadata) Enqueue(msg *core.Message) error {
	metadata := msg.TryGetMetadata()
	if metadata == nil {
		return router.Broadcast.Enqueue(msg)
	}

	streamName := metadata.GetValueString(router.key)
	targetID := core.GetStreamID(streamName)
	if targetID == core.InvalidStreamID {
		return router.Broadcast.Enqueue(msg)
	}

	targetRouter := core.StreamRegistry.GetRouterOrFallback(targetID)

	msg.SetStreamID(targetID)
	return core.Route(msg, targetRouter)
}
