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
	"hash/fnv"
	"sync"
)

// streamRegistry holds routers mapped by their MessageStreamID as well as a
// reverse lookup of MessageStreamID to stream name.
type streamRegistry struct {
	routers     map[MessageStreamID]Router
	name        map[MessageStreamID]string
	nameGuard   *sync.Mutex
	streamGuard *sync.Mutex
	wildcard    []Producer
}

// StreamRegistry is the global instance of streamRegistry used to store the
// all registered routers.
var StreamRegistry = streamRegistry{
	routers:     make(map[MessageStreamID]Router),
	streamGuard: new(sync.Mutex),
	name:        make(map[MessageStreamID]string),
	nameGuard:   new(sync.Mutex),
}

// GetStreamID is deprecated
func GetStreamID(stream string) MessageStreamID {
	return StreamRegistry.GetStreamID(stream)
}

// GetStreamID returns the integer representation of a given stream name.
func (registry *streamRegistry) GetStreamID(stream string) MessageStreamID {
	hash := fnv.New64a()
	hash.Write([]byte(stream))
	streamID := MessageStreamID(hash.Sum64())

	registry.nameGuard.Lock()
	defer registry.nameGuard.Unlock()
	registry.name[streamID] = stream

	return streamID
}

// GetStreamName does a reverse lookup for a given MessageStreamID and returns
// the corresponding name. If the MessageStreamID is not registered, an empty
// string is returned.
func (registry streamRegistry) GetStreamName(streamID MessageStreamID) string {
	switch streamID {
	case LogInternalStreamID:
		return LogInternalStream

	case WildcardStreamID:
		return WildcardStream

	default:
		registry.nameGuard.Lock()
		defer registry.nameGuard.Unlock()
		if name, exists := registry.name[streamID]; exists {
			return name // ### return, found ###
		}
	}
	return ""
}

// GetRouterByStreamName returns a registered stream by name. See GetRouter.
func (registry streamRegistry) GetRouterByStreamName(name string) Router {
	streamID := registry.GetStreamID(name)
	return registry.GetRouter(streamID)
}

// GetRouter returns a registered stream or nil
func (registry streamRegistry) GetRouter(id MessageStreamID) Router {
	registry.streamGuard.Lock()
	defer registry.streamGuard.Unlock()
	stream, exists := registry.routers[id]
	if !exists {
		return nil
	}
	return stream
}

// IsStreamRegistered returns true if the stream for the given id is registered.
func (registry streamRegistry) IsStreamRegistered(id MessageStreamID) bool {
	registry.streamGuard.Lock()
	defer registry.streamGuard.Unlock()
	_, exists := registry.routers[id]
	return exists
}

// ForEachStream loops over all registered routers and calls the given function.
func (registry streamRegistry) ForEachStream(callback func(streamID MessageStreamID, stream Router)) {
	registry.streamGuard.Lock()
	routers := registry.routers
	registry.streamGuard.Unlock()

	for streamID, router := range routers {
		callback(streamID, router)
	}
}

// WildcardProducersExist returns true if any producer is listening to the
// wildcard stream.
func (registry *streamRegistry) WildcardProducersExist() bool {
	return len(registry.wildcard) > 0
}

// RegisterWildcardProducer adds a new producer to the list of known wildcard
// prodcuers. This list has to be added to new routers upon creation to send
// messages to producers listening to *.
// Duplicates will be filtered.
// This state of this list is undefined during the configuration phase.
func (registry *streamRegistry) RegisterWildcardProducer(producers ...Producer) {
nextProd:
	for _, prod := range producers {
		for _, existing := range registry.wildcard {
			if existing == prod {
				continue nextProd
			}
		}
		registry.wildcard = append(registry.wildcard, prod)
	}
}

// AddWildcardProducersToRouter adds all known wildcard producers to a given
// router. The state of the wildcard list is undefined during the configuration
// phase.
func (registry streamRegistry) AddWildcardProducersToRouter(router Router) {
	streamID := router.GetStreamID()
	if streamID != LogInternalStreamID {
		router.AddProducer(registry.wildcard...)
	}
}

// AddAllWildcardProducersToAllRouters executes AddWildcardProducersToRouter on
// all currently registered routers
func (registry *streamRegistry) AddAllWildcardProducersToAllRouters() {
	registry.ForEachStream(
		func(streamID MessageStreamID, router Router) {
			registry.AddWildcardProducersToRouter(router)
		})
}

// Register registers a router plugin to a given stream id
func (registry *streamRegistry) Register(router Router, streamID MessageStreamID) {
	registry.streamGuard.Lock()
	defer registry.streamGuard.Unlock()

	if _, exists := registry.routers[streamID]; exists {
		tlog.Warning.Printf("%T attaches to an already occupied router (%s)", router, registry.GetStreamName(streamID))
	} else {
		CountRouters()
	}
	registry.routers[streamID] = router
}

func (registry *streamRegistry) createFallback(streamID MessageStreamID) Router {
	streamName := registry.GetStreamName(streamID)
	tlog.Debug.Print("Creating fallback stream for ", streamName)

	config := NewPluginConfig("_generated_stream_"+streamName, "router.Broadcast")
	config.Override("stream", streamName)

	plugin, err := NewPluginWithConfig(config)
	if err != nil {
		panic(err) // this has to always work, otherwise: panic
	}

	stream := plugin.(Router) // panic if not!
	return stream
}

// GetRouterOrFallback returns the router for the given streamID if it is registered.
// If no router is registered for the given streamID the default router is used.
// The default router is equivalent to an unconfigured router.Broadcast with
// all wildcard producers already added.
func (registry *streamRegistry) GetRouterOrFallback(streamID MessageStreamID) Router {
	if streamID == InvalidStreamID {
		return nil
	}

	registry.streamGuard.Lock()
	defer registry.streamGuard.Unlock()

	if router, exists := registry.routers[streamID]; exists {
		return router
	}

	defaultRouter := registry.createFallback(streamID)
	registry.AddWildcardProducersToRouter(defaultRouter)

	registry.routers[streamID] = defaultRouter

	CountRouters()
	if streamID != InvalidStreamID && streamID != WildcardStreamID {
		CountFallbackRouters()
	}

	return defaultRouter
}
