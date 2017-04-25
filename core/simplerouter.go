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

package core

import (
	"github.com/trivago/tgo/tlog"
	"time"
)

// SimpleRouter router plugin
// Configuration example:
//
//  MyRouter:
//    Type: "router.Broadcast"
//    Stream: "foo"
//    TimeoutMs: 200
//    Modulators:
//	- filter.RegExp:
// 	  Expression: "[a-zA-Z]+"
//
type SimpleRouter struct {
	id         string
	modulators ModulatorArray
	Producers  []Producer
	Timeout    *time.Duration
	streamID   MessageStreamID
	Log        tlog.LogScope
}

// Configure sets up all values required by SimpleRouter.
func (router *SimpleRouter) Configure(conf PluginConfigReader) error {
	router.id = conf.GetID()
	router.Log = conf.GetLogScope()
	router.Timeout = nil
	router.streamID = conf.GetStreamID("Stream", GetStreamID(conf.GetID()))
	router.modulators = conf.GetModulatorArray("Modulators", router.Log, ModulatorArray{})

	if router.streamID == WildcardStreamID {
		router.Log.Note.Print("A wildcard stream configuration only affects the wildcard stream, not all routers")
	}

	if conf.HasValue("TimeoutMs") {
		timeout := time.Duration(conf.GetInt("TimeoutMs", 0)) * time.Millisecond
		router.Timeout = &timeout
	}

	return conf.Errors.OrNil()
}

// GetID returns the ID of this stream
func (router *SimpleRouter) GetID() string {
	return router.id
}

// StreamID returns the id of the stream this plugin is bound to.
func (router *SimpleRouter) StreamID() MessageStreamID {
	return router.streamID
}

// AddProducer adds all producers to the list of known producers.
// Duplicates will be filtered.
func (router *SimpleRouter) AddProducer(producers ...Producer) {
	for _, prod := range producers {
		for _, inListProd := range router.Producers {
			if inListProd == prod {
				return // ### return, already in list ###
			}
		}
		router.Producers = append(router.Producers, prod)
	}
}

// GetProducers returns the producers bound to this stream
func (router *SimpleRouter) GetProducers() []Producer {
	return router.Producers
}

// Modulate calls all modulators in their order of definition
func (router *SimpleRouter) Modulate(msg *Message) ModulateResult {
	return router.modulators.Modulate(msg)
}
