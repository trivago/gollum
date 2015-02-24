package main

import (
	"fmt"
	"github.com/trivago/gollum/log"
	"github.com/trivago/gollum/shared"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
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
func newMultiplexer(configFile string, profile bool) multiplexer {
	conf, err := shared.ReadConfig(configFile)
	if err != nil {
		fmt.Printf("Error: %s", err.Error())
		os.Exit(-1)
	}

	// Configure the multiplexer, create a byte pool and assign it to the log

	logConsumer := Log.Consumer{}
	logConsumer.Configure(shared.PluginConfig{})

	plex := multiplexer{
		stream:          make(map[shared.MessageStreamID][]shared.Producer),
		managedStream:   make(map[shared.MessageStreamID][]shared.Distributor),
		consumerThreads: new(sync.WaitGroup),
		producerThreads: new(sync.WaitGroup),
		consumers:       []shared.Consumer{logConsumer},
		profile:         profile,
	}

	// Initialize the plugins based on the config

	for className, instanceConfigs := range conf.Settings {
		for _, config := range instanceConfigs {
			if !config.Enable {
				continue // ### continue, disabled ###
			}

			// Try to instantiate and configure the plugin

			plugin, err := shared.RuntimeType.NewPlugin(className, config)
			if err != nil {
				if plugin == nil {
					Log.Error.Panic(err.Error())
				} else {
					Log.Error.Print("Failed to configure ", className, ": ", err)
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
	}

	return plex
}

// sendMessage is the default distributor which sends a pinned message to all
// producers in the list.
func (plex multiplexer) sendMessage(message shared.Message, producers []shared.Producer, sendToInactive bool) {
	for _, producer := range producers {
		shared.SingleDistribute(producer, message, sendToInactive)
	}
}

// distribute pinns the message to a specific stream and forwards it to either
// all producers registered to that stream or to all distributors registered to
// that stream.
func (plex multiplexer) distribute(message shared.Message, streamID shared.MessageStreamID, sendToInactive bool) {
	producers := plex.stream[streamID]
	if len(producers) > 0 {
		pinnedMsg := message.CloneAndPin(streamID)
		distributors, isManaged := plex.managedStream[streamID]

		if isManaged {
			for _, distributor := range distributors {
				distributor.Distribute(pinnedMsg, producers, sendToInactive)
			}
		} else {
			plex.sendMessage(pinnedMsg, producers, sendToInactive)
		}
	}
}

// broadcastMessage does the initial distribution of the message, i.e. it takes
// care of sending messages to the wildcardstream and to all other streams.
func (plex multiplexer) broadcastMessage(message shared.Message, sendToInactive bool) {
	// Send to wildcard stream producers if not purely internal
	if !message.IsInternal() {
		plex.distribute(message, shared.WildcardStreamID, sendToInactive)
	}

	// Send to specific stream producers
	for _, streamID := range message.Streams {
		plex.distribute(message, streamID, sendToInactive)
	}
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
	defer plex.shutdown()

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

	// Launch producers
	for _, producer := range plex.producers {
		go producer.Produce(plex.producerThreads)
	}

	// If there are intenal log listeners switch to stream mode
	if _, enableQueue := plex.stream[shared.LogInternalStreamID]; enableQueue {
		Log.EnqueueMessages(true)
	}

	// Launch consumers
	for _, consumer := range plex.consumers {
		go consumer.Consume(plex.consumerThreads)
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

		if plex.profile {
			duration := time.Since(measure)
			if messageCount >= 100000 || duration.Seconds() > 5 {
				Log.Note.Printf("Processed %.2f msg/sec", float64(messageCount)/duration.Seconds())

				measure = time.Now()
				messageCount = 0
			}
		}
	}
}
