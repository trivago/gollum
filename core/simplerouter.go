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

package core

import (
	"github.com/sirupsen/logrus"
	"github.com/trivago/tgo/thealthcheck"
	"strings"
	"time"
)

// SimpleRouter plugin base type
//
// This type defines a common baseclass for routers. All routers should
// derive from this class, but not necessarily need to.
//
// Parameters
//
// - Stream: This value specifies the name of the stream this plugin is supposed to
// read messages from.
//
// - Filters: This value defines an optional list of Filter plugins to connect to
// this router.
//
// - TimeoutMs: This value sets a timeout in milliseconds until a message should
// handled by the router. You can disable this behavior by setting it to "0".
// By default this parameter is set to "0".
//
type SimpleRouter struct {
	id        string
	Producers []Producer
	filters   FilterArray     `config:"Filters"`
	timeout   time.Duration   `config:"TimeoutMs" default:"0" metric:"ms"`
	streamID  MessageStreamID `config:"Stream"`
	Logger    logrus.FieldLogger
}

// Configure sets up all values required by SimpleRouter.
func (router *SimpleRouter) Configure(conf PluginConfigReader) {
	router.id = conf.GetID()
	router.Logger = conf.GetLogger()

	if router.streamID == WildcardStreamID && strings.Index(router.id, GeneratedRouterPrefix) != 0 {
		router.Logger.Info("A wildcard stream configuration only affects the wildcard stream, not all routers")
	}
}

// GetLogger returns the logging scope of this plugin
func (router *SimpleRouter) GetLogger() logrus.FieldLogger {
	return router.Logger
}

// AddHealthCheck adds a health check at the default URL (http://<addr>:<port>/<plugin_id>)
func (router *SimpleRouter) AddHealthCheck(callback thealthcheck.CallbackFunc) {
	router.AddHealthCheckAt("", callback)
}

// AddHealthCheckAt adds a health check at a subpath (http://<addr>:<port>/<plugin_id><path>)
func (router *SimpleRouter) AddHealthCheckAt(path string, callback thealthcheck.CallbackFunc) {
	thealthcheck.AddEndpoint("/"+router.GetID()+path, callback)
}

// GetID returns the ID of this router
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
