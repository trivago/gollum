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
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	metricMsgSec    = "MessagesPerSec"
	metricMsgSecAvg = "MessagesPerSecAvg"
	metricCons      = "Consumers"
	metricProds     = "Producers"
	metricStreams   = "ManagedStreams"
	metricMessages  = "Messages"
)

type multiplexerState byte

const (
	multiplexerStateConfigure      = multiplexerState(iota)
	multiplexerStateStartProducers = multiplexerState(iota)
	multiplexerStateStartConsumers = multiplexerState(iota)
	multiplexerStateRunning        = multiplexerState(iota)
	multiplexerStateShutdown       = multiplexerState(iota)
	multiplexerStateStopConsumers  = multiplexerState(iota)
	multiplexerStateStopProducers  = multiplexerState(iota)
	multiplexerStateStopped        = multiplexerState(iota)
)

type multiplexer struct {
	consumers      []core.Consumer
	producers      []core.Producer
	streams        map[core.MessageStreamID][]core.Distributor
	consumerWorker *sync.WaitGroup
	producerWorker *sync.WaitGroup
	state          multiplexerState
	signal         chan os.Signal
	profile        bool
}

// Create a new multiplexer based on a given config file.
func newMultiplexer(conf *core.Config, profile bool) multiplexer {
	// Configure the multiplexer, create a byte pool and assign it to the log

	logConsumer := core.LogConsumer{}
	logConsumer.Configure(core.PluginConfig{})

	shared.Metric.New(metricMsgSec)
	shared.Metric.New(metricMsgSecAvg)
	shared.Metric.New(metricCons)
	shared.Metric.New(metricProds)
	shared.Metric.New(metricStreams)
	shared.Metric.New(metricMessages)

	plex := multiplexer{
		consumers:      []core.Consumer{&logConsumer},
		streams:        make(map[core.MessageStreamID][]core.Distributor),
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

		plugin, err := core.NewPlugin(config)
		if err != nil {
			if plugin == nil {
				Log.Error.Panic(err.Error())
			} else {
				Log.Error.Print("Failed to configure ", config.TypeName, ": ", err)
				continue // ### continue ###
			}
		}

		// Register consumer plugins
		if consumer, isConsumer := plugin.(core.Consumer); isConsumer {
			plex.consumers = append(plex.consumers, consumer)
			shared.Metric.Inc(metricCons)

			for i := 1; i < config.Instances; i++ {
				clone, _ := core.NewPlugin(config)
				plex.consumers = append(plex.consumers, clone.(core.Consumer))
				shared.Metric.Inc(metricCons)
			}
		}

		// Register producer plugins
		if producer, isProducer := plugin.(core.Producer); isProducer {
			plex.producers = append(plex.producers, producer)
			shared.Metric.Inc(metricProds)

			for i := 1; i < config.Instances; i++ {
				clone, _ := core.NewPlugin(config)
				plex.producers = append(plex.producers, clone.(core.Producer))
				shared.Metric.Inc(metricCons)
			}
		}

		// Register dsitributor plugins
		if _, isDistributor := plugin.(core.Distributor); isDistributor {
			for _, stream := range config.Stream {
				streamID := core.GetStreamID(stream)

				// New instance per stream
				distributor, _ := core.NewPlugin(config)

				if stream, isMapped := plex.streams[streamID]; isMapped {
					plex.streams[streamID] = append(stream, distributor.(core.Distributor))
				} else {
					plex.streams[streamID] = []core.Distributor{distributor.(core.Distributor)}
				}
			}
		}
	}

	// Analyze all producers and add them to the corresponding distributors
	// Wildcard streams will be handled after this so we have a fully configured
	// map to add to.

	var wildcardProducers []core.Producer

	for _, prod := range plex.producers {
		// Check all streams for each producer
		for _, streamID := range prod.Streams() {
			if streamID == core.WildcardStreamID {
				wildcardProducers = append(wildcardProducers, prod)
			}

			if distList, isMapped := plex.streams[streamID]; isMapped {
				// Add to all distributors for this stream
				for _, dist := range distList {
					dist.AddProducer(prod)
				}
			} else {
				// No distributor for this stream: Create broadcast
				// distributor as a default.
				defaultDistPlugin, _ := shared.RuntimeType.New("distributor.Broadcast")
				defaultDist := defaultDistPlugin.(core.Distributor)
				defaultDist.AddProducer(prod)
				plex.streams[streamID] = []core.Distributor{defaultDist}
			}
		}
	}

	// Append wildcard producers to all streams

	for _, prod := range wildcardProducers {
		for streamID, distList := range plex.streams {
			switch streamID {
			case core.WildcardStreamID:
			case core.DroppedStreamID:
			case core.LogInternalStreamID:
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
	// Make Ctrl+C possible during shutdown sequence
	if plex.signal != nil {
		signal.Stop(plex.signal)
	}

	Log.Note.Print("Filthy little hobbites. They stole it from us. (shutdown)")
	stateAtShutdown := plex.state

	// Shutdown consumers
	plex.state = multiplexerStateStopConsumers
	if stateAtShutdown >= multiplexerStateStartConsumers {
		for _, consumer := range plex.consumers {
			consumer.Control() <- core.ConsumerControlStop
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
	Log.SetWriter(os.Stdout)

	// Shutdown producers
	plex.state = multiplexerStateStopProducers
	if stateAtShutdown >= multiplexerStateStartProducers {
		for _, producer := range plex.producers {
			producer.Control() <- core.ProducerControlStop
		}
		plex.producerWorker.Wait()
	}

	// Done
	plex.state = multiplexerStateStopped
}

func (plex *multiplexer) distribute(msg core.Message) {
	for _, streamID := range msg.Streams {
		msg.CurrentStream = streamID

		distList, isMapped := plex.streams[streamID]
		if !isMapped {
			distList = plex.streams[core.WildcardStreamID]
		}

		for _, distributor := range distList {
			distributor.Distribute(msg)
		}
	}
}

func (plex *multiplexer) handleSignals() {
	plex.signal = make(chan os.Signal, 1)
	signal.Notify(plex.signal, os.Interrupt, syscall.SIGHUP)

	for {
		sig := <-plex.signal
		switch sig {
		case syscall.SIGINT:
			Log.Note.Print("Master betrayed us. Wicked. Tricksy, False. (signal)")
			plex.state = multiplexerStateShutdown
			return // ### return, exit requested ###

		case syscall.SIGHUP:
			for _, consumer := range plex.consumers {
				consumer.Control() <- core.ConsumerControlRoll
			}
			for _, producer := range plex.producers {
				producer.Control() <- core.ProducerControlRoll
			}
		}
	}
}

func (plex *multiplexer) handlePanics() {
	if r := recover(); r != nil {
		log.Println(r)
	}

	plex.shutdown()
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

	defer plex.handlePanics()

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
	if _, enableQueue := plex.streams[core.LogInternalStreamID]; enableQueue {
		Log.SetWriter(plex.consumers[0].(*core.LogConsumer))
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
	go plex.handleSignals()
	Log.Note.Print("We be nice to them, if they be nice to us. (startup)")

	measure := time.Now()
	messageCount := 0
	plex.state = multiplexerStateRunning

	for plex.state != multiplexerStateShutdown {
		// Go over all consumers in round-robin fashion
		// Don't block here, too as a consumer might not contain new messages

		messageCountBefore := messageCount

		for _, consumer := range plex.consumers {
			select {
			default:
			case message := <-consumer.Messages():
				plex.distribute(message)
				messageCount++
			}
		}

		// Sleep if there is nothing to do

		if messageCount-messageCountBefore == 0 {
			time.Sleep(time.Duration(100) * time.Millisecond)
		}

		// Profiling information

		duration := time.Since(measure)
		if messageCount >= 100000 || duration.Seconds() > 5 {
			// Local values
			value := float64(messageCount) / duration.Seconds()
			shared.Metric.SetF(metricMsgSec, value)
			shared.Metric.AddI(metricMessages, messageCount)

			if plex.profile {
				Log.Note.Printf("Processed %.2f msg/sec", value)
			}

			// Global values
			timeSinceStart := time.Since(shared.ProcessStartTime)
			if totalMessages, err := shared.Metric.Get(metricMessages); err == nil {
				value = float64(totalMessages) / timeSinceStart.Seconds()
				shared.Metric.SetF(metricMsgSecAvg, value)
			}

			// Prepare next run
			measure = time.Now()
			messageCount = 0
		}
	}
}
