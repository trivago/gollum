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
	"github.com/trivago/tgo"
	"github.com/trivago/tgo/tlog"
	"github.com/trivago/tgo/treflect"
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
		tlog.SetWriter(os.Stdout)
	})
	defer logFallback.Stop()

	// Configure the multiplexer, create a byte pool and assign it to the log
	tgo.Metric.New(metricCons)
	tgo.Metric.New(metricProds)
	tgo.Metric.New(metricMessages)
	tgo.Metric.New(metricDropped)
	tgo.Metric.New(metricDiscarded)
	tgo.Metric.New(metricNoRoute)
	tgo.Metric.New(metricFiltered)
	tgo.Metric.New(metricMessagesSec)
	tgo.Metric.New(metricDroppedSec)
	tgo.Metric.New(metricDiscardedSec)
	tgo.Metric.New(metricNoRouteSec)
	tgo.Metric.New(metricFilteredSec)
	tgo.Metric.New(metricBlockedProducers)
	tgo.Metric.New(metricVersion)

	if gollumDevVer > 0 {
		tgo.Metric.Set(metricVersion, gollumMajorVer*1000000+gollumMinorVer*10000+gollumPatchVer*100+gollumDevVer)
	} else {
		tgo.Metric.Set(metricVersion, gollumMajorVer*10000+gollumMinorVer*100+gollumPatchVer)
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

		if config.Typename == "" {
			tlog.Error.Printf("Failed to load plugin %s. Type not set", config.ID)
			continue // ### continue ###
		}

		pluginType := core.TypeRegistry.GetTypeOf(config.Typename)
		if pluginType == nil {
			tlog.Error.Printf("Failed to load plugin %s. Type %s not found", config.ID, config.Typename)
			continue // ### continue ###
		}

		tlog.Debug.Print("Loading plugin ", config.ID)

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
		plugin, err := core.NewPlugin(config)
		if err != nil {
			tlog.Error.Printf("Failed to configure stream %s: %s", config.ID, err)
			continue // ### continue ###
		}

		streamPlugin := plugin.(core.Stream)

		tlog.Debug.Printf("Configuring %s (%s) as %s", config.ID, core.StreamRegistry.GetStreamName(streamPlugin.GetBoundStreamID()), config.Typename)
		core.StreamRegistry.Register(streamPlugin, streamPlugin.GetBoundStreamID())
	}

	// All producers are added to the wildcard stream so that consumers can send
	// to all producers if required. The wildcard producer list is required
	// to add producers listening to all streams to all streams that are used.

	wildcardStream := core.StreamRegistry.GetStreamOrFallback(core.WildcardStreamID)

	for _, config := range producerConfig {
		for i := 0; i < config.Instances; i++ {
			tlog.Debug.Print("Configuring ", config.Typename)
			plugin, err := core.NewPlugin(config)

			if err != nil {
				tlog.Error.Print("Failed to configure producer plugin ", config.Typename, ": ", err)
				continue // ### continue ###
			}

			producer, _ := plugin.(core.Producer)
			streams := producer.Streams()

			if len(streams) == 0 {
				tlog.Error.Print("Producer plugin ", config.Typename, " has no streams set")
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
			tgo.Metric.Inc(metricProds)
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
	logConsumerConfig := core.NewPluginConfig("", "core.LogConsumer")
	logConsumer.Configure(core.NewPluginConfigReader(&logConsumerConfig))

	for _, config := range consumerConfig {
		for i := 0; i < config.Instances; i++ {
			tlog.Debug.Print("Configuring ", config.Typename)
			plugin, err := core.NewPlugin(config)
			if err != nil {
				tlog.Error.Print("Failed to configure consumer plugin ", config.Typename, ": ", err)
				continue // ### continue ###
			}

			consumer, _ := plugin.(core.Consumer)
			plex.consumers = append(plex.consumers, consumer)
			tgo.Metric.Inc(metricCons)
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
	tlog.Error.Print("Failed to load plugin ", typeName, ": Does not qualify for consumer, producer or stream interface")

	consumerMatch, consumerMissing := treflect.GetMissingMethods(pluginType, consumerInterface)
	producerMatch, producerMissing := treflect.GetMissingMethods(pluginType, producerInterface)
	streamMatch, streamMissing := treflect.GetMissingMethods(pluginType, streamInterface)

	if consumerMatch > producerMatch {
		if consumerMatch > streamMatch {
			tlog.Error.Print("Plugin looks like a consumer:")
			for _, message := range consumerMissing {
				tlog.Error.Print(message)
			}
		} else {
			tlog.Error.Print("Plugin looks like a stream:")
			for _, message := range streamMissing {
				tlog.Error.Print(message)
			}
		}
	} else if producerMatch > streamMatch {
		tlog.Error.Print("Plugin looks like a producer:")
		for _, message := range producerMissing {
			tlog.Error.Print(message)
		}
	} else {
		tlog.Error.Print("Plugin looks like a stream:")
		for _, message := range streamMissing {
			tlog.Error.Print(message)
		}
	}
}

// Shutdown all consumers and producers in a clean way.
// The internal log is flushed after the consumers have been shut down so that
// consumer related messages are still in the tlog.
// Producers are flushed after flushing the log, so producer related shutdown
// messages will be posted to stdout
func (plex *multiplexer) shutdown() {
	// Make sure the log is printed to stdout if we are stuck here
	logFallback := time.AfterFunc(time.Duration(3)*time.Second, func() {
		tlog.SetWriter(os.Stdout)
	})

	defer func() {
		if r := recover(); r != nil {
			tlog.Error.Println(r)
			tlog.Error.Print(string(debug.Stack()))
		}
		logFallback.Stop()
	}()

	// Make Ctrl+C possible during shutdown sequence
	if plex.signal != nil {
		signal.Stop(plex.signal)
	}

	tlog.Note.Print("Filthy little hobbites. They stole it from us. (shutdown)")
	stateAtShutdown := plex.state

	// Shutdown consumers
	plex.state = multiplexerStateStopConsumers
	if stateAtShutdown >= multiplexerStateStartConsumers {
		for _, cons := range plex.consumers {
			tlog.Debug.Printf("Closing consumer %s", reflect.TypeOf(cons).String())
			cons.Control() <- core.PluginControlStopConsumer
		}
		core.StreamRegistry.ActivateAllFuses()
		tlog.Debug.Print("Waiting for consumers to close")
		plex.consumerWorker.Wait()
	}

	// Make sure remaining warning / errors are written to stderr
	tlog.Note.Print("It's the only way. Go in, or go back. (flushing)")
	logFallback.Stop()
	tlog.SetWriter(os.Stdout)

	// Shutdown producers
	plex.state = multiplexerStateStopProducers
	if stateAtShutdown >= multiplexerStateStartProducers {
		for _, prod := range plex.producers {
			tlog.Debug.Printf("Closing producer %s", reflect.TypeOf(prod).String())
			prod.Control() <- core.PluginControlStopProducer
		}
		tlog.Debug.Print("Waiting for producers to close")
		plex.producerWorker.Wait()
	}

	plex.state = multiplexerStateStopped
}

// Run the multiplexer.
// Fetch messags from the consumers and pass them to all producers.
func (plex multiplexer) run() {
	// Log consumer is always active
	if len(plex.consumers) == 0 {
		tlog.Error.Print("No consumers configured.")
		tlog.SetWriter(os.Stdout)
		return // ### return, nothing to do ###
	}

	if len(plex.producers) == 0 {
		tlog.Error.Print("No producers configured.")
		tlog.SetWriter(os.Stdout)
		return // ### return, nothing to do ###
	}

	defer plex.shutdown()

	// Launch producers
	plex.state = multiplexerStateStartProducers
	for _, producer := range plex.producers {
		producer := producer
		tlog.Debug.Print("Starting ", reflect.TypeOf(producer))
		go tgo.WithRecoverShutdown(func() {
			producer.Produce(plex.producerWorker)
		})
	}

	// If there are intenal log listeners switch to stream mode
	if core.StreamRegistry.IsStreamRegistered(core.LogInternalStreamID) {
		tlog.Debug.Print("Binding log to ", reflect.TypeOf(plex.consumers[0]))
		tlog.SetWriter(plex.consumers[0].(*core.LogConsumer))
	} else {
		tlog.SetWriter(os.Stdout)
	}

	// Launch consumers
	plex.state = multiplexerStateStartConsumers
	for _, consumer := range plex.consumers {
		consumer := consumer
		tlog.Debug.Print("Starting ", reflect.TypeOf(consumer))
		go tgo.WithRecoverShutdown(func() {
			consumer.Consume(plex.consumerWorker)
		})
	}

	// Main loop - wait for exit
	// Apache is using SIG_USR1 in some cases to signal child processes.
	// This signal is not available on windows

	plex.signal = newSignalHandler()

	tlog.Note.Print("We be nice to them, if they be nice to us. (startup)")
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

			tgo.Metric.SetF(metricMessagesSec, messageSec)
			tgo.Metric.SetF(metricDroppedSec, float64(droppedCount)/duration.Seconds())
			tgo.Metric.SetF(metricDiscardedSec, float64(discardedCount)/duration.Seconds())
			tgo.Metric.SetF(metricFilteredSec, float64(filteredCount)/duration.Seconds())
			tgo.Metric.SetF(metricNoRouteSec, float64(noRouteCount)/duration.Seconds())

			tgo.Metric.Add(metricMessages, int64(messageCount))
			tgo.Metric.Add(metricDropped, int64(droppedCount))
			tgo.Metric.Add(metricDiscarded, int64(discardedCount))
			tgo.Metric.Add(metricFiltered, int64(filteredCount))
			tgo.Metric.Add(metricNoRoute, int64(noRouteCount))

			if plex.profile {
				tlog.Note.Printf("Processed %.2f msg/sec", messageSec)
			}

			// Blocked producers
			numBlockedProducers := 0
			for _, prod := range plex.producers {
				if prod.IsBlocked() {
					numBlockedProducers++
				}
			}
			tgo.Metric.SetI(metricBlockedProducers, numBlockedProducers)

		case sig := <-plex.signal:
			switch translateSignal(sig) {
			case signalExit:
				tlog.Note.Print("Master betrayed us. Wicked. Tricksy, False. (signal)")
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
