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
	"reflect"
	"sync"
	"time"
)

const (
	metricMsgSec    = "MessagesPerSec"
	metricMsgSecAvg = "MessagesPerSecAvg"
	metricCons      = "Consumers"
	metricProds     = "Producers"
	metricMessages  = "Messages"
)

type multiplexerState byte
type signalType byte

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

const (
	signalNone = signalType(iota)
	signalExit = signalType(iota)
	signalRoll = signalType(iota)
)

type multiplexer struct {
	consumers      []core.Consumer
	producers      []core.Producer
	consumerWorker *sync.WaitGroup
	producerWorker *sync.WaitGroup
	state          multiplexerState
	signal         chan os.Signal
	profile        bool
}

// Create a new multiplexer based on a given config file.
func newMultiplexer(conf *core.Config, profile bool) multiplexer {
	// Configure the multiplexer, create a byte pool and assign it to the log

	shared.Metric.New(metricMsgSec)
	shared.Metric.New(metricMsgSecAvg)
	shared.Metric.New(metricCons)
	shared.Metric.New(metricProds)
	shared.Metric.New(metricMessages)

	plex := multiplexer{
		consumers:      []core.Consumer{new(core.LogConsumer)},
		consumerWorker: new(sync.WaitGroup),
		producerWorker: new(sync.WaitGroup),
		profile:        profile,
		state:          multiplexerStateConfigure,
	}

	// Sort the plugins by interface type.

	var consumerConfig, producerConfig, streamConfig []core.PluginConfig
	consumerInterface := reflect.TypeOf((*core.Consumer)(nil)).Elem()
	producerInterface := reflect.TypeOf((*core.Producer)(nil)).Elem()
	streamInterface := reflect.TypeOf((*core.Stream)(nil)).Elem()

	for _, config := range conf.Plugins {
		if !config.Enable {
			continue // ### continue, disabled ###
		}

		pluginType := shared.RuntimeType.GetTypeOf(config.Typename)
		if pluginType == nil {
			Log.Error.Print("Failed to load plugin ", config.Typename, ": Type not found")
			continue // ### continue ###
		}

		validPlugin := false
		if pluginType.Implements(consumerInterface) {
			consumerConfig = append(consumerConfig, config)
			validPlugin = true
		}
		if pluginType.Implements(producerInterface) {
			producerConfig = append(producerConfig, config)
			validPlugin = true
		}
		if pluginType.Implements(streamInterface) {
			streamConfig = append(streamConfig, config)
			validPlugin = true
		}

		if !validPlugin {
			Log.Error.Print("Failed to load plugin ", config.Typename, ": Does not qualify for consumer, producer or stream interface")

			consumerMatch, consumerMissing := shared.GetMissingMethods(pluginType, consumerInterface)
			producerMatch, producerMissing := shared.GetMissingMethods(pluginType, producerInterface)
			streamMatch, streamMissing := shared.GetMissingMethods(pluginType, streamInterface)

			if consumerMatch > producerMatch {
				if consumerMatch > streamMatch {
					Log.Error.Print("Plugin looks like a consumer:")
					for _, message := range consumerMissing {
						Log.Error.Print(message)
					}
				} else {
					Log.Error.Print("Plugin looks like a stream:")
					for _, message := range streamMissing {
						Log.Error.Print(message)
					}
				}
			} else if producerMatch > streamMatch {
				Log.Error.Print("Plugin looks like a producer:")
				for _, message := range producerMissing {
					Log.Error.Print(message)
				}
			} else {
				Log.Error.Print("Plugin looks like a stream:")
				for _, message := range streamMissing {
					Log.Error.Print(message)
				}
			}
		}
	}

	// Initialize the plugins in the order of stream, producer, consumer to
	// match the order of reference between the different types.

	for _, config := range streamConfig {
		for _, streamName := range config.Stream {
			plugin, err := core.NewPlugin(config)
			if err != nil {
				Log.Error.Print("Failed to configure stream plugin ", config.Typename, ": ", err)
				continue // ### continue ###
			}
			core.StreamTypes.Register(plugin.(core.Stream), core.GetStreamID(streamName))
		}
	}

	// All producers are added to the wildcard stream so that consumers can send
	// to all producers if required. The wildcard producer list is required
	// to add producers listening to all streams to all streams that are used.

	wildcardStream := core.StreamTypes.GetStreamOrFallback(core.WildcardStreamID)

	for _, config := range producerConfig {
		for i := 0; i < config.Instances; i++ {
			plugin, err := core.NewPlugin(config)
			if err != nil {
				Log.Error.Print("Failed to configure producer plugin ", config.Typename, ": ", err)
				continue // ### continue ###
			}

			producer, _ := plugin.(core.Producer)
			streams := producer.Streams()

			if len(streams) == 0 {
				Log.Error.Print("Producer plugin ", config.Typename, " has no streams set")
				continue // ### continue ###
			}

			for _, streamID := range streams {
				if streamID == core.WildcardStreamID {
					core.StreamTypes.RegisterWildcardProducer(producer)
				} else {
					stream := core.StreamTypes.GetStreamOrFallback(streamID)
					stream.AddProducer(producer)
				}
			}

			// Do not add internal streams to wildcard stream

			for _, streamID := range streams {
				if streamID != core.LogInternalStreamID && streamID != core.DroppedStreamID {
					wildcardStream.AddProducer(producer)
					break
				}
			}

			plex.producers = append(plex.producers, producer)
			shared.Metric.Inc(metricProds)
		}
	}

	// Consumers are registered last so that the stream reference list can be
	// built. This eliminates lookups when sending to specific streams.

	logConsumer, _ := plex.consumers[0].(*core.LogConsumer)
	logConsumer.Configure(core.NewPluginConfig("core.LogConsumer"))

	for _, config := range consumerConfig {
		for i := 0; i < config.Instances; i++ {
			plugin, err := core.NewPlugin(config)
			if err != nil {
				Log.Error.Print("Failed to configure consumer plugin ", config.Typename, ": ", err)
				continue // ### continue ###
			}

			consumer, _ := plugin.(core.Consumer)
			plex.consumers = append(plex.consumers, consumer)
			shared.Metric.Inc(metricCons)
		}
	}

	// As consumers might create new fallback streams this is the first position
	// where we can add the wildcard producers to all streams. No new streams
	// created beyond this point must use StreamRegistry.AddWildcardProducersToStream.

	core.StreamTypes.ForEachStream(
		func(streamID core.MessageStreamID, stream core.Stream) {
			switch streamID {
			case core.LogInternalStreamID, core.WildcardStreamID, core.DroppedStreamID:
				// Internal streams are excluded for wildcard listeners
			default:
				core.StreamTypes.AddWildcardProducersToStream(stream)
			}
		})

	return plex
}

// Shutdown all consumers and producers in a clean way.
// The internal log is flushed after the consumers have been shut down so that
// consumer related messages are still in the log.
// Producers are flushed after flushing the log, so producer related shutdown
// messages will be posted to stdout
func (plex *multiplexer) shutdown() {
	// Handle panics if any
	if r := recover(); r != nil {
		log.Println(r)
	}

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
			consumer.Control() <- core.PluginControlStop
		}

		plex.consumerWorker.Wait()
	}

	// Make sure remaining warning / errors are written to stderr
	Log.SetWriter(os.Stdout)
	Log.Note.Print("It's the only way. Go in, or go back. (flushing)")

	// Shutdown producers
	plex.state = multiplexerStateStopProducers
	if stateAtShutdown >= multiplexerStateStartProducers {
		for _, producer := range plex.producers {
			producer.Control() <- core.PluginControlStop
		}
		plex.producerWorker.Wait()
	}

	plex.state = multiplexerStateStopped
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

	defer plex.shutdown()

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
	if core.StreamTypes.IsStreamRegistered(core.LogInternalStreamID) {
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

	// Main loop - wait for exit
	// Apache is using SIG_USR1 in some cases to signal child processes.
	// This signal is not available on windows

	plex.signal = newSignalHandler()

	Log.Note.Print("We be nice to them, if they be nice to us. (startup)")
	measure := time.Now()
	timer := time.NewTicker(time.Duration(2) * time.Second)

	for {
		select {
		case <-timer.C:
			duration := time.Since(measure)
			measure = time.Now()

			// Sampling based values
			messageCount := core.GetAndResetMessageCount()
			value := float64(messageCount) / duration.Seconds()
			shared.Metric.SetF(metricMsgSec, value)
			shared.Metric.Add(metricMessages, int64(messageCount))

			if plex.profile {
				Log.Note.Printf("Processed %.2f msg/sec", value)
			}

			// Global values
			timeSinceStart := time.Since(shared.ProcessStartTime)
			if totalMessages, err := shared.Metric.Get(metricMessages); err == nil {
				value = float64(totalMessages) / timeSinceStart.Seconds()
				shared.Metric.SetF(metricMsgSecAvg, value)
			}

		case sig := <-plex.signal:
			switch translateSignal(sig) {
			case signalExit:
				Log.Note.Print("Master betrayed us. Wicked. Tricksy, False. (signal)")
				plex.state = multiplexerStateShutdown
				return // ### return, exit requested ###

			case signalRoll:
				for _, consumer := range plex.consumers {
					consumer.Control() <- core.PluginControlRoll
				}
				for _, producer := range plex.producers {
					producer.Control() <- core.PluginControlRoll
				}

			default:
			}
		}
	}
}
