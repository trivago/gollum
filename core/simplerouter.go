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
	"github.com/trivago/tgo/thealthcheck"
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
//    Filters:
//	- filter.RegExp:
// 	  Expression: "[a-zA-Z]+"
//
type SimpleRouter struct {
	id        string
	Producers []Producer
	filters   FilterArray     `config:"Filters"`
	timeout   time.Duration   `config:"TimeoutMs" default:"0" metric:"ms"`
	streamID  MessageStreamID `config:"Stream"`
	Log       tlog.LogScope
}

// Configure sets up all values required by SimpleRouter.
func (router *SimpleRouter) Configure(conf PluginConfigReader) error {
	conf.Configure(router, router.Log)

	router.id = conf.GetID()
	router.Log = conf.GetLogScope()
	//router.Timeout = nil
	//router.streamID = conf.GetStreamID("Stream", GetStreamID(conf.GetID()))

	//router.filters = conf.GetFilterArray("Filters", router.Log, FilterArray{})

	if router.streamID == WildcardStreamID {
		router.Log.Note.Print("A wildcard stream configuration only affects the wildcard stream, not all routers")
	}

	//if conf.HasValue("TimeoutMs") {
	//	timeout := time.Duration(conf.GetInt("TimeoutMs", 0)) * time.Millisecond
	//	router.Timeout = &timeout
	//}

	return conf.Errors.OrNil()
}

// AddHealthCheck adds a health check at the default URL (http://<addr>:<port>/<plugin_id>)
func (router *SimpleRouter) AddHealthCheck(callback thealthcheck.CallbackFunc) {
	router.AddHealthCheckAt("", callback)
}

// AddHealthCheckAt adds a health check at a subpath (http://<addr>:<port>/<plugin_id><path>)
func (router *SimpleRouter) AddHealthCheckAt(path string, callback thealthcheck.CallbackFunc) {
	thealthcheck.AddEndpoint("/"+router.GetID()+path, callback)
}

// GetID returns the ID of this stream
func (router *SimpleRouter) GetID() string {
	return router.id
}

// GetStreamID returns the id of the stream this plugin is bound to.
func (router *SimpleRouter) GetStreamID() MessageStreamID {
	return router.streamID
}

// GetTimeout returns the timeout set for this router
func (router *SimpleRouter) GetTimeout() time.Duration {
	return router.timeout
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
	mod := NewFilterModulator(router.filters)
	return mod.Modulate(msg)
}
