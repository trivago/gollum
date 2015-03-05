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
	"github.com/trivago/gollum/shared"
	"sync"
)

// Loop consumer plugin for processing feedback messages
// Configuration example
//
//   - "consumer.Loop":
//     Enable: true
//     Routes:
//        "myStream" : "myOtherStream"
//        "_DROPPED_" : "myStream"
//
// This consumer ignores the "Stream" setting.
//
// Routes defines a 1:1 stream remapping. Messages reaching the Loop consumer
// over the feedback channel are rewritten to be sent to the given stream
// mapping. Messages that have been dropped due to a timeout are automatically
// sent to the "_DROPPED_" stream. If nothing is configured this stream will
// be the routing target for all streams.
// The default route is set to "*" : "_DROPPED_". Note that in this case "*"
// includes internal streams, too.
type Loop struct {
	shared.ConsumerBase
	quit   bool
	routes map[shared.MessageStreamID]shared.MessageStreamID
}

func init() {
	shared.RuntimeType.Register(Loop{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *Loop) Configure(conf shared.PluginConfig) error {
	err := cons.ConsumerBase.Configure(conf)
	if err != nil {
		return err
	}

	cons.routes = conf.GetStreamRoute("Routes", shared.DroppedStreamID)
	shared.EnableFeedbackQueue(conf.Channel)

	return nil
}

func (cons *Loop) route(msg shared.Message) {
	if route, exists := cons.routes[msg.CurrentStream]; exists {
		msg.Streams = []shared.MessageStreamID{route}
	} else {
		msg.Streams = []shared.MessageStreamID{cons.routes[shared.WildcardStreamID]}
	}

	cons.PostMessage(msg)
}

func (cons *Loop) stop() {
	defer cons.MarkAsDone()

	// Flush
	for {
		select {
		case msg := <-shared.GetFeedbackQueue():
			cons.route(msg)
		default:
			return // ### return, done ###
		}
	}
}

func (cons *Loop) processFeedbackQueue() {
	defer cons.stop()

	for !cons.quit {
		msg := <-shared.GetFeedbackQueue()
		cons.route(msg)
	}
}

// Consume is fetching and forwarding messages from the feedbackQueue
func (cons Loop) Consume(threads *sync.WaitGroup) {
	cons.quit = false

	go func() {
		defer shared.RecoverShutdown()
		cons.processFeedbackQueue()
	}()

	defer func() {
		cons.quit = true
	}()

	cons.DefaultControlLoop(threads, nil)
}
