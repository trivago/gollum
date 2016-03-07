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

package main

import (
	"container/list"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	"log"
	"os"
	"os/signal"
	"reflect"
	"runtime/debug"
	"sync"
	"time"
)

const (
	metricMessagesSec      = "MessagesPerSec"
	metricCons             = "Consumers"
	metricProds            = "Producers"
	metricMessages         = "Messages"
	metricDiscarded        = "DiscardedMessages"
	metricDropped          = "DroppedMessages"
	metricFiltered         = "Filtered"
	metricNoRoute          = "DiscardedNoRoute"
	metricDiscardedSec     = "DiscardedMessagesSec"
	metricDroppedSec       = "DroppedMessagesSec"
	metricFilteredSec      = "FilteredSec"
	metricNoRouteSec       = "DiscardedNoRouteSec"
	metricBlockedProducers = "BlockedProducers"
	metricVersion          = "Version"
)

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

var (
	consumerInterface = reflect.TypeOf((*core.Consumer)(nil)).Elem()
	producerInterface = reflect.TypeOf((*core.Producer)(nil)).Elem()
	streamInterface   = reflect.TypeOf((*core.Stream)(nil)).Elem()
)

type multiplexerState byte
type signalType byte

type multiplexer struct {
	consumers      []core.Consumer
	producers      []core.Producer
	shutdownOrder  *list.List
	consumerWorker *sync.WaitGroup
	producerWorker *sync.WaitGroup
	state          multiplexerState
	signal         chan os.Signal
	profile        bool
}

// Create a new multiplexer based on a given config file.
func newMultiplexer(conf *core.Config, profile bool) multiplexer {
	// Make sure the log is printed to stdout if we are stuck here
	logFallback := time.AfterFunc(time.Duration(3)*time.Second, func() {
		Log.SetWriter(os.Stdout)
	})
	defer logFallback.Stop()

	// Configure the multiplexer, create a byte pool and assign it to the log
	shared.Metric.New(metricCons)
	shared.Metric.New(metricProds)
	shared.Metric.New(metricMessages)
	shared.Metric.New(metricDropped)
	shared.Metric.New(metricDiscarded)
	shared.Metric.New(metricNoRoute)
	shared.Metric.New(metricFiltered)
	shared.Metric.New(metricMessagesSec)
	shared.Metric.New(metricDroppedSec)
	shared.Metric.New(metricDiscardedSec)
	shared.Metric.New(metricNoRouteSec)
	shared.Metric.New(metricFilteredSec)
	shared.Metric.New(metricBlockedProducers)
	shared.Metric.New(metricVersion)

	if gollumDevVer > 0 {
		shared.Metric.Set(metricVersion, gollumMajorVer*1000000+gollumMinorVer*10000+gollumPatchVer*100+gollumDevVer)
	} else {
		shared.Metric.Set(metricVersion, gollumMajorVer*10000+gollumMinorVer*100+gollumPatchVer)
	}
	plex := multiplexer{
		consumers:      []core.Consumer{new(core.LogConsumer)},
		consumerWorker: new(sync.WaitGroup),
		producerWorker: new(sync.WaitGroup),
		shutdownOrder:  list.New(),
		profile:        profile,
		state:          multiplexerStateConfigure,
	}

	// Sort the plugins by interface type.

	var consumerConfig, producerConfig, streamConfig []core.PluginConfig

	for _, config := range conf.Plugins {
		if !config.Enable {
			continue // ### continue, disabled ###
		}

		Log.Debug.Print("Loading ", config.Typename)

		pluginType := shared.TypeRegistry.GetTypeOf(config.Typename)
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
			dumpFaultyPlugin(config.Typename, pluginType)
		}
	}

	// Initialize the plugins in the order of stream, producer, consumer to
	// match the order of reference between the different types.

	for _, config := range streamConfig {
		if len(config.Stream) == 0 {
			Log.Error.Printf("Stream plugin %s has no streams set", config.Typename)
			continue // ### continue ###
		}

		streamName := config.Stream[0]
		if len(config.Stream) > 1 {
			Log.Warning.Printf("Stream plugins may only bind to one stream. Plugin will bind to %s", streamName)
		}

		plugin, err := core.NewPlugin(config)
		if err != nil {
			Log.Error.Printf("Failed to configure stream %s: %s", streamName, err)
			continue // ### continue ###
		}

		Log.Debug.Print("Configuring ", config.Typename, " for ", streamName)
		core.StreamRegistry.Register(plugin.(core.Stream), core.StreamRegistry.GetStreamID(streamName))
	}

	// All producers are added to the wildcard stream so that consumers can send
	// to all producers if required. The wildcard producer list is required
	// to add producers listening to all streams to all streams that are used.

	wildcardStream := core.StreamRegistry.GetStreamOrFallback(core.WildcardStreamID)

	for _, config := range producerConfig {
		for i := 0; i < config.Instances; i++ {
			Log.Debug.Print("Configuring ", config.Typename)
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
					core.StreamRegistry.RegisterWildcardProducer(producer)
				} else {
					stream := core.StreamRegistry.GetStreamOrFallback(streamID)
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

	// Register dependencies by going over each producer and registering it to
	// all producers listening to its DropStream

	for _, prod := range plex.producers {
		core.StreamRegistry.LinkDependencies(prod, prod.GetDropStreamID())
	}

	// Consumers are registered last so that the stream reference list can be
	// built. This eliminates lookups when sending to specific streams.

	logConsumer, _ := plex.consumers[0].(*core.LogConsumer)
	logConsumer.Configure(core.NewPluginConfig("core.LogConsumer"))

	for _, config := range consumerConfig {
		for i := 0; i < config.Instances; i++ {
			Log.Debug.Print("Configuring ", config.Typename)
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

	core.StreamRegistry.ForEachStream(
		func(streamID core.MessageStreamID, stream core.Stream) {
			core.StreamRegistry.AddWildcardProducersToStream(stream)
		})

	return plex
}

func dumpFaultyPlugin(typeName string, pluginType reflect.Type) {
	Log.Error.Print("Failed to load plugin ", typeName, ": Does not qualify for consumer, producer or stream interface")

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

// Shutdown all consumers and producers in a clean way.
// The internal log is flushed after the consumers have been shut down so that
// consumer related messages are still in the log.
// Producers are flushed after flushing the log, so producer related shutdown
// messages will be posted to stdout
func (plex *multiplexer) shutdown() {
	// Make sure the log is printed to stdout if we are stuck here
	logFallback := time.AfterFunc(time.Duration(3)*time.Second, func() {
		Log.SetWriter(os.Stdout)
	})

	defer func() {
		if r := recover(); r != nil {
			log.Println(r)
			log.Print(string(debug.Stack()))
		}
		logFallback.Stop()
	}()

	// Make Ctrl+C possible during shutdown sequence
	if plex.signal != nil {
		signal.Stop(plex.signal)
	}

	Log.Note.Print("Filthy little hobbites. They stole it from us. (shutdown)")
	stateAtShutdown := plex.state

	// Shutdown consumers
	plex.state = multiplexerStateStopConsumers
	if stateAtShutdown >= multiplexerStateStartConsumers {
		for _, cons := range plex.consumers {
			Log.Debug.Printf("Closing consumer %s", reflect.TypeOf(cons).String())
			cons.Control() <- core.PluginControlStopConsumer
		}
		core.StreamRegistry.ActivateAllFuses()
		Log.Debug.Print("Waiting for consumers to close")
		plex.consumerWorker.Wait()
	}

	// Make sure remaining warning / errors are written to stderr
	Log.Note.Print("It's the only way. Go in, or go back. (flushing)")
	logFallback.Stop()
	Log.SetWriter(os.Stdout)

	// Shutdown producers
	plex.state = multiplexerStateStopProducers
	if stateAtShutdown >= multiplexerStateStartProducers {
		for _, prod := range plex.producers {
			Log.Debug.Printf("Closing producer %s", reflect.TypeOf(prod).String())
			prod.Control() <- core.PluginControlStopProducer
		}
		Log.Debug.Print("Waiting for producers to close")
		plex.producerWorker.Wait()
	}

	plex.state = multiplexerStateStopped
}

// Run the multiplexer.
// Fetch messags from the consumers and pass them to all producers.
func (plex multiplexer) run() {
	if len(plex.consumers) == 0 {
		Log.Error.Print("No consumers configured.")
		Log.SetWriter(os.Stdout)
		return // ### return, nothing to do ###
	}

	if len(plex.producers) == 0 {
		Log.Error.Print("No producers configured.")
		Log.SetWriter(os.Stdout)
		return // ### return, nothing to do ###
	}

	defer plex.shutdown()

	// Launch producers
	plex.state = multiplexerStateStartProducers
	for _, producer := range plex.producers {
		producer := producer
		Log.Debug.Print("Starting ", reflect.TypeOf(producer))
		go shared.DontPanic(func() {
			producer.Produce(plex.producerWorker)
		})
	}

	// If there are intenal log listeners switch to stream mode
	if core.StreamRegistry.IsStreamRegistered(core.LogInternalStreamID) {
		Log.Debug.Print("Binding log to ", reflect.TypeOf(plex.consumers[0]))
		Log.SetWriter(plex.consumers[0].(*core.LogConsumer))
	} else {
		Log.SetWriter(os.Stdout)
	}

	// Launch consumers
	plex.state = multiplexerStateStartConsumers
	for _, consumer := range plex.consumers {
		consumer := consumer
		Log.Debug.Print("Starting ", reflect.TypeOf(consumer))
		go shared.DontPanic(func() {
			consumer.Consume(plex.consumerWorker)
		})
	}

	// Main loop - wait for exit
	// Apache is using SIG_USR1 in some cases to signal child processes.
	// This signal is not available on windows

	plex.signal = newSignalHandler()

	Log.Note.Print("We be nice to them, if they be nice to us. (startup)")
	measure := time.Now()
	timer := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-timer.C:
			duration := time.Since(measure)
			measure = time.Now()

			// Sampling values
			messageCount, droppedCount, discardedCount, filteredCount, noRouteCount := core.GetAndResetMessageCount()
			messageSec := float64(messageCount) / duration.Seconds()

			shared.Metric.SetF(metricMessagesSec, messageSec)
			shared.Metric.SetF(metricDroppedSec, float64(droppedCount)/duration.Seconds())
			shared.Metric.SetF(metricDiscardedSec, float64(discardedCount)/duration.Seconds())
			shared.Metric.SetF(metricFilteredSec, float64(filteredCount)/duration.Seconds())
			shared.Metric.SetF(metricNoRouteSec, float64(noRouteCount)/duration.Seconds())

			shared.Metric.Add(metricMessages, int64(messageCount))
			shared.Metric.Add(metricDropped, int64(droppedCount))
			shared.Metric.Add(metricDiscarded, int64(discardedCount))
			shared.Metric.Add(metricFiltered, int64(filteredCount))
			shared.Metric.Add(metricNoRoute, int64(noRouteCount))

			if plex.profile {
				Log.Note.Printf("Processed %.2f msg/sec", messageSec)
			}

			// Blocked producers
			numBlockedProducers := 0
			for _, prod := range plex.producers {
				if prod.IsBlocked() {
					numBlockedProducers++
				}
			}
			shared.Metric.SetI(metricBlockedProducers, numBlockedProducers)

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
