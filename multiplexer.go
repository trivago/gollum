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

package main

import (
	"github.com/trivago/gollum/log"
	"github.com/trivago/gollum/shared"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	metricMsgSec  = "MessagesPerSec"
	metricCons    = "Consumers"
	metricProds   = "Producers"
	metricStreams = "ManagedStreams"
)

type multiplexer struct {
	consumers       []shared.Consumer
	producers       []shared.Producer
	consumerThreads *sync.WaitGroup
	producerThreads *sync.WaitGroup
	stream          map[shared.MessageStreamID][]shared.Producer
	managedStream   map[shared.MessageStreamID][]shared.Distributor
	profile         bool
}

// Create a new multiplexer based on a given config file.
func newMultiplexer(conf *shared.Config, profile bool) multiplexer {
	// Configure the multiplexer, create a byte pool and assign it to the log

	logConsumer := Log.Consumer{}
	logConsumer.Configure(shared.PluginConfig{})

	Log.Metric.New(metricMsgSec)
	Log.Metric.New(metricCons)
	Log.Metric.New(metricProds)
	Log.Metric.New(metricStreams)

	plex := multiplexer{
		stream:          make(map[shared.MessageStreamID][]shared.Producer),
		managedStream:   make(map[shared.MessageStreamID][]shared.Distributor),
		consumerThreads: new(sync.WaitGroup),
		producerThreads: new(sync.WaitGroup),
		consumers:       []shared.Consumer{logConsumer},
		profile:         profile,
	}

	// Initialize the plugins based on the config

	for _, config := range conf.Plugins {
		if !config.Enable {
			continue // ### continue, disabled ###
		}

		// Try to instantiate and configure the plugin

		plugin, err := shared.RuntimeType.NewPlugin(config)
		if err != nil {
			if plugin == nil {
				Log.Error.Panic(err.Error())
			} else {
				Log.Error.Print("Failed to configure ", config.TypeName, ": ", err)
				continue // ### continue ###
			}
		}

		// Register dsitributor plugins
		if distributor, isDistributor := plugin.(shared.Distributor); isDistributor {
			for _, stream := range config.Stream {
				streamID := shared.GetStreamID(stream)
				streamMap, mappingExists := plex.managedStream[streamID]

				if !mappingExists {
					plex.managedStream[streamID] = []shared.Distributor{distributor}
				} else {
					plex.managedStream[streamID] = append(streamMap, distributor)
				}
			}
		}

		// Register consumer plugins
		if consumer, isConsumer := plugin.(shared.Consumer); isConsumer {
			plex.consumers = append(plex.consumers, consumer)
		}

		// Register producer plugins
		if producer, isProducer := plugin.(shared.Producer); isProducer {
			plex.producers = append(plex.producers, producer)

			for _, stream := range config.Stream {
				streamID := shared.GetStreamID(stream)
				streamMap, mappingExists := plex.stream[streamID]

				if !mappingExists {
					plex.stream[streamID] = []shared.Producer{producer}
				} else {
					plex.stream[streamID] = append(streamMap, producer)
				}
			}
		}
	}

	Log.Metric.SetI(metricCons, len(plex.consumers))
	Log.Metric.SetI(metricProds, len(plex.producers))
	Log.Metric.SetI(metricStreams, len(plex.managedStream))

	return plex
}

// distribute pinns the message to a specific stream and forwards it to either
// all producers registered to that stream or to all distributors registered to
// that stream.
func (plex multiplexer) distribute(message shared.Message, streamID shared.MessageStreamID, sendToInactive bool) {
	producers := plex.stream[streamID]
	message.SetCurrentStream(streamID)

	if distributors, isManaged := plex.managedStream[streamID]; isManaged {
		for _, distributor := range distributors {
			distributor.Distribute(message, producers, sendToInactive)
		}
	} else {
		for _, producer := range producers {
			shared.SingleDistribute(producer, message, sendToInactive)
		}
	}
}

// broadcastMessage does the initial distribution of the message, i.e. it takes
// care of sending messages to the wildcardstream and to all other streams.
func (plex multiplexer) broadcastMessage(message shared.Message, sendToInactive bool) {
	// Send to wildcard stream producers if not purely internal
	if !message.IsInternalOnly() {
		plex.distribute(message, shared.WildcardStreamID, sendToInactive)
	}

	// Send to specific stream producers
	message.ForEachStream(func(streamID shared.MessageStreamID) bool {
		plex.distribute(message, streamID, sendToInactive)
		return true
	})
}

// Shutdown all consumers and producers in a clean way.
// The internal log is flushed after the consumers have been shut down so that
// consumer related messages are still in the log.
// Producers are flushed after flushing the log, so producer related shutdown
// messages will be posted to stdout
func (plex *multiplexer) shutdown() {
	Log.Note.Print("Filthy little hobbites. They stole it from us. (shutdown)")

	// Send shutdown to consumers
	for _, consumer := range plex.consumers {
		consumer.Control() <- shared.ConsumerControlStop
	}

	// Make sure all remaining messages are flushed BEFORE waiting for all
	// consumers to stop. This is necessary as consumers might be waiting in
	// a push to channel.
	Log.Note.Print("It's the only way. Go in, or go back. (flushing)")

	for _, consumer := range plex.consumers {
	flushing:
		for {
			select {
			case message := <-consumer.Messages():
				// A flush may happen before any producers are started. In that
				// case we need to ignore these producers.
				plex.broadcastMessage(message, false)
			default:
				break flushing
			}
		}
	}

	plex.consumerThreads.Wait()

	// Make sure remaining warning / errors are written to stderr
	Log.EnqueueMessages(false)

	// Shutdown producers
	for _, producer := range plex.producers {
		producer.Control() <- shared.ProducerControlStop
	}
	plex.producerThreads.Wait()
}

// Run the multiplexer.
// Fetch messags from the consumers and pass them to all producers.
func (plex multiplexer) run() {
	if len(plex.consumers) == 0 {
		Log.Error.Print("No consumers configured.")
		return // ### return, nothing to do ###
	}

	if len(plex.producers) == 0 {
		Log.Error.Print("No producers configured.")
		return // ### return, nothing to do ###
	}

	// React on signals and setup the MessageProvider queue
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGHUP)

	defer func() {
		if r := recover(); r != nil {
			log.Print("PANIC: ", r)
		}
		signal.Stop(signalChannel)
		plex.shutdown()
	}()

	// Launch producers
	for _, producer := range plex.producers {
		producer := producer
		go func() {
			defer shared.RecoverShutdown()
			producer.Produce(plex.producerThreads)
		}()
	}

	// If there are intenal log listeners switch to stream mode
	if _, enableQueue := plex.stream[shared.LogInternalStreamID]; enableQueue {
		Log.EnqueueMessages(true)
	}

	// Launch consumers
	for _, consumer := range plex.consumers {
		consumer := consumer
		go func() {
			defer shared.RecoverShutdown()
			consumer.Consume(plex.consumerThreads)
		}()
	}

	// Main loop
	Log.Note.Print("We be nice to them, if they be nice to us. (startup)")

	measure := time.Now()
	messageCount := 0

	for {
		// Go over all consumers in round-robin fashion
		// Don't block here, too as a consumer might not contain new messages

		for _, consumer := range plex.consumers {
			select {
			default:
				// do nothing

			case sig := <-signalChannel:
				switch sig {
				case syscall.SIGINT:
					Log.Note.Print("Master betrayed us. Wicked. Tricksy, False. (signal)")
					return

				case syscall.SIGHUP:
					for _, consumer := range plex.consumers {
						consumer.Control() <- shared.ConsumerControlRoll
					}
					for _, producer := range plex.producers {
						producer.Control() <- shared.ProducerControlRoll
					}
				}

			case message := <-consumer.Messages():
				plex.broadcastMessage(message, true)
				messageCount++
			}
		}

		duration := time.Since(measure)
		if messageCount >= 100000 || duration.Seconds() > 5 {
			value := float64(messageCount) / duration.Seconds()
			if plex.profile {
				Log.Note.Printf("Processed %.2f msg/sec", value)
			}
			Log.Metric.SetF(metricMsgSec, value)
			measure = time.Now()
			messageCount = 0
		}
	}
}
