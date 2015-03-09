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
	metricMsgSec   = "MessagesPerSec"
	metricCons     = "Consumers"
	metricProds    = "Producers"
	metricStreams  = "ManagedStreams"
	metricMessages = "Messages"
)

type multiplexerState byte

const (
	multiplexerStateConfigure      = multiplexerState(iota)
	multiplexerStateStartProducers = multiplexerState(iota)
	multiplexerStateStartConsumers = multiplexerState(iota)
	multiplexerStateRunning        = multiplexerState(iota)
	multiplexerStateStopConsumers  = multiplexerState(iota)
	multiplexerStateStopProducers  = multiplexerState(iota)
	multiplexerStateStopped        = multiplexerState(iota)
)

type multiplexer struct {
	consumers      []shared.Consumer
	producers      []shared.Producer
	streams        map[shared.MessageStreamID][]shared.Distributor
	consumerWorker *sync.WaitGroup
	producerWorker *sync.WaitGroup
	state          multiplexerState
	profile        bool
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
	Log.Metric.New(metricMessages)

	plex := multiplexer{
		consumers:      []shared.Consumer{logConsumer},
		streams:        make(map[shared.MessageStreamID][]shared.Distributor),
		consumerWorker: new(sync.WaitGroup),
		producerWorker: new(sync.WaitGroup),
		profile:        profile,
		state:          multiplexerStateConfigure,
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

		// Register consumer plugins
		if consumer, isConsumer := plugin.(shared.Consumer); isConsumer {
			plex.consumers = append(plex.consumers, consumer)
			Log.Metric.Inc(metricCons)
		}

		// Register producer plugins
		if producer, isProducer := plugin.(shared.Producer); isProducer {
			plex.producers = append(plex.producers, producer)
			Log.Metric.Inc(metricProds)
		}

		// Register dsitributor plugins
		if _, isDistributor := plugin.(shared.Distributor); isDistributor {
			for _, stream := range config.Stream {
				streamID := shared.GetStreamID(stream)

				// New instance per stream
				distributor, _ := shared.RuntimeType.NewPlugin(config)

				if stream, isMapped := plex.streams[streamID]; isMapped {
					plex.streams[streamID] = append(stream, distributor.(shared.Distributor))
				} else {
					plex.streams[streamID] = []shared.Distributor{distributor.(shared.Distributor)}
				}
			}
		}
	}

	// Analyze all producers and add them to the corresponding distributors
	// Wildcard streams will be handled after this so we have a fully configured
	// map to add to.

	var wildcardProducers []shared.Producer

	for _, prod := range plex.producers {
		// Check all streams for each producer
		for _, streamID := range prod.Streams() {
			if streamID == shared.WildcardStreamID {
				// Wildcard stream handling
				wildcardProducers = append(wildcardProducers, prod)
			} else {
				if distList, isMapped := plex.streams[streamID]; isMapped {
					// Add to all distributors for this stream
					for _, dist := range distList {
						dist.AddProducer(prod)
					}
				} else {
					// No distributor for this stream: Create broadcast
					// distributor as a default.
					defaultDistPlugin, _ := shared.RuntimeType.New("distributor.Broadcast")
					defaultDist := defaultDistPlugin.(shared.Distributor)
					defaultDist.AddProducer(prod)
					plex.streams[streamID] = []shared.Distributor{defaultDist}
				}
			}
		}
	}

	// Append wildcard producers to all streams

	for _, prod := range wildcardProducers {
		for streamID, distList := range plex.streams {
			switch streamID {
			case shared.WildcardStreamID:
			case shared.DroppedStreamID:
			case shared.LogInternalStreamID:
			default:
				for _, dist := range distList {
					dist.AddProducer(prod)
				}
			}
		}
	}

	return plex
}

// Shutdown all consumers and producers in a clean way.
// The internal log is flushed after the consumers have been shut down so that
// consumer related messages are still in the log.
// Producers are flushed after flushing the log, so producer related shutdown
// messages will be posted to stdout
func (plex *multiplexer) shutdown() {
	Log.Note.Print("Filthy little hobbites. They stole it from us. (shutdown)")

	stateAtShutdown := plex.state

	// Shutdown consumers
	plex.state = multiplexerStateStopConsumers
	if stateAtShutdown >= multiplexerStateStartConsumers {
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
					plex.distribute(message)
				default:
					break flushing
				}
			}
		}

		plex.consumerWorker.Wait()
	}

	// Make sure remaining warning / errors are written to stderr
	Log.EnqueueMessages(false)

	// Shutdown producers
	plex.state = multiplexerStateStopProducers
	if stateAtShutdown >= multiplexerStateStartProducers {
		for _, producer := range plex.producers {
			producer.Control() <- shared.ProducerControlStop
		}
		plex.producerWorker.Wait()
	}

	// Done
	plex.state = multiplexerStateStopped
}

func (plex multiplexer) distribute(msg shared.Message) {
	for _, streamID := range msg.Streams {
		distList := plex.streams[streamID]
		msg.CurrentStream = streamID

		for _, distributor := range distList {
			distributor.Distribute(msg)
		}
	}
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
	plex.state = multiplexerStateStartProducers
	for _, producer := range plex.producers {
		producer := producer
		go func() {
			defer shared.RecoverShutdown()
			producer.Produce(plex.producerWorker)
		}()
	}

	// If there are intenal log listeners switch to stream mode
	if _, enableQueue := plex.streams[shared.LogInternalStreamID]; enableQueue {
		Log.EnqueueMessages(true)
	}

	// Launch consumers
	plex.state = multiplexerStateStartConsumers
	for _, consumer := range plex.consumers {
		consumer := consumer
		go func() {
			defer shared.RecoverShutdown()
			consumer.Consume(plex.consumerWorker)
		}()
	}

	// Main loop
	plex.state = multiplexerStateRunning
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
				plex.distribute(message)
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
			Log.Metric.AddI(metricMessages, messageCount)

			measure = time.Now()
			messageCount = 0
		}
	}
}
