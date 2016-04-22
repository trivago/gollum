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
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo"
	"github.com/trivago/tgo/tlog"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"time"
)

const (
	metricCons  = "Consumers"
	metricProds = "Producers"
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

type multiplexerState byte
type signalType byte

// Multiplexer is the main gollum instance taking care of starting and stopping
// plugins.
type Multiplexer struct {
	consumers      []core.Consumer
	producers      []core.Producer
	consumerWorker *sync.WaitGroup
	producerWorker *sync.WaitGroup
	logConsumer    *core.LogConsumer
	state          multiplexerState
	signal         chan os.Signal
}

// NewMultiplexer creates a new multplexer
func NewMultiplexer() Multiplexer {
	tgo.Metric.New(metricCons)
	tgo.Metric.New(metricProds)

	return Multiplexer{
		consumerWorker: new(sync.WaitGroup),
		producerWorker: new(sync.WaitGroup),
		state:          multiplexerStateConfigure,
	}
}

// Configure processes the config and instantiates all valid plugins
func (plex *Multiplexer) Configure(conf *core.Config) {
	// Make sure the log is printed to stderr if we are stuck here
	logFallback := time.AfterFunc(time.Duration(3)*time.Second, func() {
		tlog.SetWriter(os.Stderr)
	})
	defer logFallback.Stop()

	// Initialize the plugins in the order of streams > producers > consumers
	// to match the order of reference between the different types.

	plex.configureStreams(conf)
	plex.configureProducers(conf)
	plex.configureConsumers(conf)

	// As consumers might create new fallback streams this is the first position
	// where we can add the wildcard producers to all streams. No new streams
	// created beyond this point must use StreamRegistry.AddWildcardProducersToStream.

	core.StreamRegistry.AddAllWildcardProducersToAllStreams()
}

// StartPlugins starts all plugins in the correct order.
func (plex *Multiplexer) StartPlugins() {

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

	// Launch producers
	plex.state = multiplexerStateStartProducers
	for _, producer := range plex.producers {
		producer := producer
		go tgo.WithRecoverShutdown(func() {
			tlog.Debug.Print("Starting ", reflect.TypeOf(producer))
			producer.Produce(plex.producerWorker)
		})
	}

	// If there are intenal log listeners switch to stream mode
	if core.StreamRegistry.IsStreamRegistered(core.LogInternalStreamID) {
		tlog.SetWriter(plex.logConsumer)
	} else {
		tlog.SetWriter(os.Stdout)
	}

	// Launch consumers
	plex.state = multiplexerStateStartConsumers
	for _, consumer := range plex.consumers {
		consumer := consumer
		go tgo.WithRecoverShutdown(func() {
			tlog.Debug.Print("Starting ", reflect.TypeOf(consumer))
			consumer.Consume(plex.consumerWorker)
		})
	}
}

// Run is essentially the multiplexer main loop.
// It listens for shutdown signals and updates global metrics
func (plex *Multiplexer) Run() {
	plex.signal = newSignalHandler()
	defer signal.Stop(plex.signal)

	tlog.Note.Print("We be nice to them, if they be nice to us. (startup)")

	for {
		sig := <-plex.signal
		switch translateSignal(sig) {
		case signalExit:
			tlog.Note.Print("Master betrayed us. Wicked. Tricksy, False. (signal)")
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

// Shutdown all consumers and producers in a clean way.
// The internal log is flushed after the consumers have been shut down so that
// consumer related messages are still in the tlog.
// Producers are flushed after flushing the log, so producer related shutdown
// messages will be posted to stdout
func (plex *Multiplexer) Shutdown() {
	tlog.Note.Print("Filthy little hobbites. They stole it from us. (shutdown)")

	stateAtShutdown := plex.state
	plex.state = multiplexerStateShutdown

	plex.shutdownConsumers(stateAtShutdown)

	// Make sure remaining warning / errors are written to stderr
	tlog.Note.Print("I'm not listening... I'm not listening... (flushing)")
	tlog.SetWriter(os.Stdout)

	// Shutdown producers
	plex.shutdownProducers(stateAtShutdown)

	plex.state = multiplexerStateStopped
}

func (plex *Multiplexer) configureStreams(conf *core.Config) {
	streamConfigs := conf.GetStreams()
	for _, config := range streamConfigs {
		tlog.Debug.Print("Instantiating ", config.ID)

		plugin, err := core.NewPlugin(config)
		if err != nil {
			tlog.Error.Printf("Failed to instantiate stream %s: %s", config.ID, err)
			continue // ### continue ###
		}

		streamPlugin := plugin.(core.Stream)

		tlog.Debug.Printf("Instantiated %s (%s) as %s", config.ID, core.StreamRegistry.GetStreamName(streamPlugin.GetBoundStreamID()), config.Typename)
		core.StreamRegistry.Register(streamPlugin, streamPlugin.GetBoundStreamID())
	}
}

func (plex *Multiplexer) configureProducers(conf *core.Config) {
	plex.state = multiplexerStateStartProducers

	// All producers are added to the wildcard stream so that consumers can send
	// to all producers if required. The wildcard producer list is required
	// to add producers listening to all streams to all streams that are used.
	wildcardStream := core.StreamRegistry.GetStreamOrFallback(core.WildcardStreamID)
	producerConfigs := conf.GetProducers()

	for _, config := range producerConfigs {
		for i := 0; i < config.Instances; i++ {
			tlog.Debug.Print("Instantiating ", config.ID)

			plugin, err := core.NewPlugin(config)
			if err != nil {
				tlog.Error.Printf("Failed to instantiate producer %s: %s", config.ID, err)
				continue // ### continue ###
			}

			producer, _ := plugin.(core.Producer)
			streams := producer.Streams()

			if len(streams) == 0 {
				tlog.Error.Print("Producer ", config.ID, " has no streams set")
				continue // ### continue ###
			}

			plex.producers = append(plex.producers, producer)
			tgo.Metric.Inc(metricProds)

			// Attach producer to streams

			for _, streamID := range streams {
				if streamID == core.WildcardStreamID {
					core.StreamRegistry.RegisterWildcardProducer(producer)
				} else {
					stream := core.StreamRegistry.GetStreamOrFallback(streamID)
					stream.AddProducer(producer)
				}
			}

			// Add producer to wildcard stream unless it only listens to internal streams

			for _, streamID := range streams {
				switch streamID {
				case core.LogInternalStreamID, core.DroppedStreamID:
				default:
					wildcardStream.AddProducer(producer)
					return
				}
			}
		}
	}
}

func (plex *Multiplexer) configureConsumers(conf *core.Config) {
	plex.state = multiplexerStateStartConsumers
	plex.configureLogConsumer()

	consumerConfigs := conf.GetConsumers()
	for _, config := range consumerConfigs {
		for i := 0; i < config.Instances; i++ {
			tlog.Debug.Print("Instantiating ", config.ID)

			plugin, err := core.NewPlugin(config)
			if err != nil {
				tlog.Error.Printf("Failed to instantiate producer %s: %s", config.ID, err)
				continue // ### continue ###
			}

			consumer, _ := plugin.(core.Consumer)
			plex.consumers = append(plex.consumers, consumer)
			tgo.Metric.Inc(metricCons)
		}
	}
}

func (plex *Multiplexer) configureLogConsumer() {
	config := core.NewPluginConfig("", "core.LogConsumer")
	configReader := core.NewPluginConfigReader(&config)

	plex.logConsumer = new(core.LogConsumer)
	plex.logConsumer.Configure(configReader)
	plex.consumers = append(plex.consumers, plex.logConsumer)
}

func (plex *Multiplexer) shutdownConsumers(stateAtShutdown multiplexerState) {
	if stateAtShutdown >= multiplexerStateStartConsumers {
		plex.state = multiplexerStateStopConsumers
		waitTimeout := time.Duration(0)

		tlog.Debug.Print("Telling consumers of stop")
		for _, cons := range plex.consumers {
			timeout := cons.GetShutdownTimeout()
			if timeout > waitTimeout {
				waitTimeout = timeout
			}
			cons.Control() <- core.PluginControlStopConsumer
		}

		waitTimeout *= 10
		tlog.Debug.Printf("Waiting for consumers to stop. Forced shutdown after %.2f seconds.", waitTimeout.Seconds())
		if !tgo.ReturnAfter(waitTimeout, plex.consumerWorker.Wait) {
			tlog.Error.Print("At least one consumer found to be blocking.")
		}
	}
}

func (plex *Multiplexer) shutdownProducers(stateAtShutdown multiplexerState) {
	if stateAtShutdown >= multiplexerStateStartProducers {
		plex.state = multiplexerStateStopProducers
		waitTimeout := time.Duration(0)

		tlog.Debug.Print("Telling producers of stop")
		for _, prod := range plex.producers {
			timeout := prod.GetShutdownTimeout()
			if timeout > waitTimeout {
				waitTimeout = timeout
			}
			prod.Control() <- core.PluginControlStopProducer
		}

		waitTimeout *= 10
		tlog.Debug.Printf("Waiting for producers to stop. Forced shutdown after %.2f seconds.", waitTimeout.Seconds())
		if !tgo.ReturnAfter(waitTimeout, plex.producerWorker.Wait) {
			tlog.Error.Print("At least one producer found to be blocking.")
		}
	}
}
