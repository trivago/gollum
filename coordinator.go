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
	coordinatorStateConfigure      = coordinatorState(iota)
	coordinatorStateStartProducers = coordinatorState(iota)
	coordinatorStateStartConsumers = coordinatorState(iota)
	coordinatorStateRunning        = coordinatorState(iota)
	coordinatorStateShutdown       = coordinatorState(iota)
	coordinatorStateStopConsumers  = coordinatorState(iota)
	coordinatorStateStopProducers  = coordinatorState(iota)
	coordinatorStateStopped        = coordinatorState(iota)
)

const (
	signalNone = signalType(iota)
	signalExit = signalType(iota)
	signalRoll = signalType(iota)
)

type coordinatorState byte
type signalType byte

// Coordinator is the main gollum instance taking care of starting and stopping
// plugins.
type Coordinator struct {
	consumers      []core.Consumer
	producers      []core.Producer
	routers        []core.Router
	consumerWorker *sync.WaitGroup
	producerWorker *sync.WaitGroup
	logConsumer    *core.LogConsumer
	state          coordinatorState
	signal         chan os.Signal
}

// NewCoordinator creates a new multplexer
func NewCoordinator() Coordinator {
	return Coordinator{
		consumerWorker: new(sync.WaitGroup),
		producerWorker: new(sync.WaitGroup),
		state:          coordinatorStateConfigure,
	}
}

// Configure processes the config and instantiates all valid plugins
func (co *Coordinator) Configure(conf *core.Config) {
	// Make sure the log is printed to stderr if we are stuck here
	logFallback := time.AfterFunc(time.Duration(3)*time.Second, func() {
		tlog.SetWriter(os.Stderr)
	})
	defer logFallback.Stop()

	// Initialize the plugins in the order of routers > producers > consumers
	// to match the order of reference between the different types.

	co.configureRouters(conf)
	co.configureProducers(conf)
	co.configureConsumers(conf)

	// As consumers might create new fallback router this is the first position
	// where we can add the wildcard producers to all streams. No new routers
	// created beyond this point must use StreamRegistry.AddWildcardProducersToRouter.

	core.StreamRegistry.AddAllWildcardProducersToAllRouters()
}

// StartPlugins starts all plugins in the correct order.
func (co *Coordinator) StartPlugins() {

	if len(co.consumers) == 0 {
		tlog.Error.Print("No consumers configured.")
		tlog.SetWriter(os.Stdout)
		return // ### return, nothing to do ###
	}

	if len(co.producers) == 0 {
		tlog.Error.Print("No producers configured.")
		tlog.SetWriter(os.Stdout)
		return // ### return, nothing to do ###
	}

	// Launch routers
	for _, router := range co.routers {
		if err := router.Start(); err != nil {
			tlog.Error.Print("Router was not able to start from type ", reflect.TypeOf(router), ": ", err)
		} else {
			tlog.Debug.Print("Starting ", reflect.TypeOf(router))
		}
	}

	// Launch producers
	co.state = coordinatorStateStartProducers
	for _, producer := range co.producers {
		producer := producer
		go tgo.WithRecoverShutdown(func() {
			tlog.Debug.Print("Starting ", reflect.TypeOf(producer))
			producer.Produce(co.producerWorker)
		})
	}

	// If there are intenal log listeners switch to stream mode
	if core.StreamRegistry.IsStreamRegistered(core.LogInternalStreamID) {
		tlog.SetWriter(co.logConsumer)
	} else {
		tlog.SetWriter(os.Stdout)
	}

	// Launch consumers
	co.state = coordinatorStateStartConsumers
	for _, consumer := range co.consumers {
		consumer := consumer
		go tgo.WithRecoverShutdown(func() {
			tlog.Debug.Print("Starting ", reflect.TypeOf(consumer))
			consumer.Consume(co.consumerWorker)
		})
	}
}

// Run is essentially the Coordinator main loop.
// It listens for shutdown signals and updates global metrics
func (co *Coordinator) Run() {
	co.signal = newSignalHandler()
	defer signal.Stop(co.signal)

	tlog.Note.Print("We be nice to them, if they be nice to us. (startup)")

	for {
		sig := <-co.signal
		switch translateSignal(sig) {
		case signalExit:
			tlog.Note.Print("Master betrayed us. Wicked. Tricksy, False. (signal)")
			return // ### return, exit requested ###

		case signalRoll:
			for _, consumer := range co.consumers {
				consumer.Control() <- core.PluginControlRoll
			}
			for _, producer := range co.producers {
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
func (co *Coordinator) Shutdown() {
	tlog.Note.Print("Filthy little hobbites. They stole it from us. (shutdown)")

	stateAtShutdown := co.state
	co.state = coordinatorStateShutdown

	co.shutdownConsumers(stateAtShutdown)

	// Make sure remaining warning / errors are written to stderr
	tlog.Note.Print("I'm not listening... I'm not listening... (flushing)")
	tlog.SetWriter(os.Stdout)

	// Shutdown producers
	co.shutdownProducers(stateAtShutdown)

	co.state = coordinatorStateStopped
}

func (co *Coordinator) configureRouters(conf *core.Config) {
	routerConfigs := conf.GetRouters()
	for _, config := range routerConfigs {
		tlog.Debug.Printf("Instantiating router '%s'", config.ID)

		plugin, err := core.NewPluginWithConfig(config)
		if err != nil {
			tlog.Error.Printf("Failed to instantiate router %s: %s", config.ID, err)
			continue // ### continue ###
		}

		routerPlugin := plugin.(core.Router)
		co.routers = append(co.routers, routerPlugin)

		tlog.Debug.Printf("Instantiated '%s' (%s) as '%s'", config.ID, core.StreamRegistry.GetStreamName(routerPlugin.GetStreamID()), config.Typename)
		core.StreamRegistry.Register(routerPlugin, routerPlugin.GetStreamID())
	}
}

func (co *Coordinator) configureProducers(conf *core.Config) {
	co.state = coordinatorStateStartProducers

	// All producers are added to the wildcard stream so that consumers can send
	// to all producers if required. The wildcard producer list is required
	// to add producers listening to all routers to all streams that are used.
	wildcardStream := core.StreamRegistry.GetRouterOrFallback(core.WildcardStreamID)
	producerConfigs := conf.GetProducers()

	for _, config := range producerConfigs {
		for i := 0; i < config.Instances; i++ {
			tlog.Debug.Print("Instantiating ", config.ID)

			plugin, err := core.NewPluginWithConfig(config)
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

			co.producers = append(co.producers, producer)
			core.CountProducers()

			// Attach producer to streams

			for _, streamID := range streams {
				if streamID == core.WildcardStreamID {
					core.StreamRegistry.RegisterWildcardProducer(producer)
				} else {
					router := core.StreamRegistry.GetRouterOrFallback(streamID)
					router.AddProducer(producer)
				}
			}

			// Add producer to wildcard stream unless it only listens to internal streams
		searchinternal:
			for _, streamID := range streams {
				switch streamID {
				case core.LogInternalStreamID:
				default:
					wildcardStream.AddProducer(producer)
					break searchinternal
				}
			}
		}
	}
}

func (co *Coordinator) configureConsumers(conf *core.Config) {
	co.state = coordinatorStateStartConsumers
	co.configureLogConsumer()

	consumerConfigs := conf.GetConsumers()
	for _, config := range consumerConfigs {
		for i := 0; i < config.Instances; i++ {
			tlog.Debug.Print("Instantiating ", config.ID)

			plugin, err := core.NewPluginWithConfig(config)
			if err != nil {
				tlog.Error.Printf("Failed to instantiate producer %s: %s", config.ID, err)
				continue // ### continue ###
			}

			consumer, _ := plugin.(core.Consumer)
			co.consumers = append(co.consumers, consumer)
			core.CountConsumers()
		}
	}
}

func (co *Coordinator) configureLogConsumer() {
	config := core.NewPluginConfig("", "core.LogConsumer")
	configReader := core.NewPluginConfigReader(&config)

	co.logConsumer = new(core.LogConsumer)
	co.logConsumer.Configure(configReader)
	co.consumers = append(co.consumers, co.logConsumer)
}

func (co *Coordinator) shutdownConsumers(stateAtShutdown coordinatorState) {
	if stateAtShutdown >= coordinatorStateStartConsumers {
		co.state = coordinatorStateStopConsumers
		waitTimeout := time.Duration(0)

		tlog.Debug.Print("Telling consumers to stop")
		for _, cons := range co.consumers {
			timeout := cons.GetShutdownTimeout()
			if timeout > waitTimeout {
				waitTimeout = timeout
			}
			cons.Control() <- core.PluginControlStopConsumer
		}

		waitTimeout *= 10
		tlog.Debug.Printf("Waiting for consumers to stop. Forced shutdown after %.2f seconds.", waitTimeout.Seconds())
		if !tgo.ReturnAfter(waitTimeout, co.consumerWorker.Wait) {
			tlog.Error.Print("At least one consumer found to be blocking.")
		}
	}
}

func (co *Coordinator) shutdownProducers(stateAtShutdown coordinatorState) {
	if stateAtShutdown >= coordinatorStateStartProducers {
		co.state = coordinatorStateStopProducers
		waitTimeout := time.Duration(0)

		tlog.Debug.Print("Telling producers to stop")
		for _, prod := range co.producers {
			timeout := prod.GetShutdownTimeout()
			if timeout > waitTimeout {
				waitTimeout = timeout
			}
			prod.Control() <- core.PluginControlStopProducer
		}

		waitTimeout *= 10
		tlog.Debug.Printf("Waiting for producers to stop. Forced shutdown after %.2f seconds.", waitTimeout.Seconds())
		if !tgo.ReturnAfter(waitTimeout, co.producerWorker.Wait) {
			tlog.Error.Print("At least one producer found to be blocking.")
		}
	}
}
