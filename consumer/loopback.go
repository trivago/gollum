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

package consumer

import (
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/shared"
	"sync"
)

// LoopBack consumer plugin for processing retried messages
// Configuration example
//
//   - "consumer.LoopBack":
//     Enable: true
//     Channel: 8192
//     Routes:
//        "_DROPPED_": "myStream"
//        "myStream":
//			- "myOtherStream"
//          - "myOtherStream2"
//
// The loopback consumer defines the configuration for the internal loopback or
// dropped message stream. Messages that are dropped because of a channel
// timeout will arrive at the _DROPPED_ channel.
// This consumer ignores the "Stream" setting and cannot be paused.
//
// Channel defines the size of the loopback queue.
// By default this is set to 8192.
//
// Routes defines a 1:n stream remapping. Messages reaching the LoopBack
// consumer are reassigned to the given stream(s). If no Route is set the
// message will be sent to its original stream.
type LoopBack struct {
	core.ConsumerBase
	quit   bool
	routes map[core.MessageStreamID]map[core.MessageStreamID]core.Stream
}

func init() {
	shared.RuntimeType.Register(LoopBack{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *LoopBack) Configure(conf core.PluginConfig) error {
	err := cons.ConsumerBase.Configure(conf)
	if err != nil {
		return err
	}

	routes := conf.GetStreamRoutes("Routes")
	cons.routes = make(map[core.MessageStreamID]map[core.MessageStreamID]core.Stream)

	for sourceID, targetIDs := range routes {
		for _, targetID := range targetIDs {
			cons.addRoute(sourceID, targetID)
		}
	}

	core.EnableRetryQueue(conf.GetInt("Channel", 8192))
	return nil
}

func (cons *LoopBack) addRoute(sourceID core.MessageStreamID, targetID core.MessageStreamID) core.Stream {
	if _, exists := cons.routes[sourceID]; !exists {
		cons.routes[sourceID] = make(map[core.MessageStreamID]core.Stream)
	}

	targetStream := core.StreamTypes.GetStreamOrFallback(targetID)
	cons.routes[sourceID][targetID] = targetStream
	return targetStream
}

func (cons *LoopBack) route(msg core.Message) {
	if streams, routeExists := cons.routes[msg.StreamID]; routeExists {
		for targetID, targetStream := range streams {
			msg.StreamID = targetID
			targetStream.Enqueue(msg)
		}
	} else {
		// Extend the cache
		targetStream := cons.addRoute(msg.StreamID, msg.StreamID)
		targetStream.Enqueue(msg)
	}
}

func (cons *LoopBack) stop() {
	defer cons.WorkerDone()

	// Flush
	for {
		select {
		case msg := <-core.GetRetryQueue():
			cons.route(msg)
		default:
			return // ### return, done ###
		}
	}
}

func (cons *LoopBack) processFeedbackQueue() {
	defer cons.stop()
	for !cons.quit {
		msg := <-core.GetRetryQueue()
		cons.route(msg)
	}
}

// Consume is fetching and forwarding messages from the feedbackQueue
func (cons *LoopBack) Consume(workers *sync.WaitGroup) {
	cons.quit = false
	defer func() { cons.quit = true }()

	go func() {
		defer shared.RecoverShutdown()
		cons.AddMainWorker(workers)
		cons.processFeedbackQueue()
	}()

	cons.DefaultControlLoop(nil)
}
