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
	"runtime"
	"sync"
)

// LoopBack consumer plugin for processing retried messages
// Configuration example
//
//   - "consumer.LoopBack":
//     Enable: true
//     Routes:
//        "_DROPPED_": "_DROPPED_"
//        "myStream":
//			- "myOtherStream"
//          - "myOtherStream2"
//
// This consumer ignores the "Stream" setting.
//
// Routes defines a 1:1 stream remapping. Messages reaching the LoopBack consumer
// over the feedback channel are rewritten to be sent to the given stream
// mapping. Messages that have been dropped due to a timeout are automatically
// sent to the "_DROPPED_" stream. If nothing is configured this stream will
// be the routing target for all streams.
// By default the route "*" is set to "_DROPPED_". I.e. if nothing else is
// configured all retried messages are sent to the _DROPPED_ channel.
// You can overwrite the default route by setting a custom route for "*".
type LoopBack struct {
	core.ConsumerBase
	quit   bool
	routes map[core.MessageStreamID][]core.MessageStreamID
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

	cons.routes = conf.GetStreamRoute("Routes", core.DroppedStreamID)
	core.EnableRetryQueue(conf.GetInt("Channel", 8192))

	return nil
}

func (cons *LoopBack) route(msg core.Message) {
	if streams, routeExists := cons.routes[msg.Stream]; routeExists {
		for _, streamID := range streams {
			msg.Stream = streamID
			cons.ReEnqueue(msg)
		}
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
		select {
		default:
			runtime.Gosched()
		case msg := <-core.GetRetryQueue():
			cons.route(msg)
		}
	}
}

// Consume is fetching and forwarding messages from the feedbackQueue
func (cons *LoopBack) Consume(workers *sync.WaitGroup) {
	cons.quit = false

	go func() {
		defer shared.RecoverShutdown()
		cons.AddMainWorker(workers)
		cons.processFeedbackQueue()
	}()

	defer func() {
		cons.quit = true
	}()

	cons.DefaultControlLoop(nil)
}
