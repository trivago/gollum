package main

import (
	"fmt"
	"github.com/trivago/gollum/shared"
	"os"
	"os/signal"
	"reflect"
	"sync"
)

type multiplexer struct {
	consumers        []shared.Consumer
	producers        []shared.Producer
	pool             *shared.BytePool
	consumerThreads  *sync.WaitGroup
	producerThreads  *sync.WaitGroup
	stream           map[shared.MessageStreamID][]*shared.Producer
	producersStarted bool
}

// Create a new multiplexer based on a given config file.
func createMultiplexer(configFile string, pool *shared.BytePool) multiplexer {
	conf, err := shared.ReadConfig(configFile)
	if err != nil {
		fmt.Printf("Error: %s", err.Error())
		os.Exit(-1)
	}

	// Configure the multiplexer, create a byte pool and assign it to the log

	var plex multiplexer
	plex.stream = make(map[shared.MessageStreamID][]*shared.Producer)
	plex.consumerThreads = new(sync.WaitGroup)
	plex.producerThreads = new(sync.WaitGroup)
	plex.pool = pool

	// Initialize the plugins based on the config

	consumerType := reflect.TypeOf((*shared.Consumer)(nil)).Elem()
	producerType := reflect.TypeOf((*shared.Producer)(nil)).Elem()

	for className, instanceConfigs := range conf.Settings {

		for _, config := range instanceConfigs {

			if !config.Enable {
				continue // ### continue, disabled ###
			}

			plugin, pluginType, err := shared.Plugin.Create(className)
			if err != nil {
				panic(err.Error())
			}

			// Register consumer plugins

			if reflect.PtrTo(pluginType).Implements(consumerType) {
				typedPlugin := plugin.(shared.Consumer)

				instance, err := typedPlugin.Create(config, plex.pool)
				if err != nil {
					shared.Log.Error("Failed registering consumer ", className, ": ", err)
					continue // ### continue ###
				}

				plex.consumers = append(plex.consumers, instance)
			}

			// Register producer plugins

			if pluginType.Implements(producerType) {
				typedPlugin := plugin.(shared.Producer)

				instance, err := typedPlugin.Create(config)
				if err != nil {
					shared.Log.Error("Failed registering producer ", className, ": ", err)
					continue // ### continue ###
				}

				for _, stream := range config.Stream {
					streamID := shared.GetStreamID(stream)
					streamMap, exists := plex.stream[streamID]
					if !exists {
						streamMap = []*shared.Producer{&instance}
						plex.stream[streamID] = streamMap
					} else {
						plex.stream[streamID] = append(streamMap, &instance)
					}
				}

				plex.producers = append(plex.producers, instance)
			}
		}
	}

	return plex
}

func (plex multiplexer) broadcastMessage(message shared.Message) {
	// Send to wildcard stream producers (all streams except internal)
	if message.StreamID != shared.LogInternalStreamID {
		for _, producer := range plex.stream[shared.WildcardStreamID] {
			if (*producer).Accepts(message) {
				message.Data.Acquire() // Add ownership for channel
				(*producer).Messages() <- message
			}
		}
	}

	// Send to specific stream producers
	for _, producer := range plex.stream[message.StreamID] {
		if (*producer).Accepts(message) {
			message.Data.Acquire() // Add ownership for channel
			(*producer).Messages() <- message
		}
	}

	message.Data.Release() // Release channel ownership
}

// Shutdown all consumers and producers in a clean way.
// The internal log is flushed after the consumers have been shut down so that
// consumer related messages are still in the log.
// Producers are flushed after flushing the log, so producer related shutdown
// messages will be posted to stdout
func (plex multiplexer) shutdown() {
	shared.Log.Note("Filthy little hobbites. They stole it from us. (shutdown)")

	// Shutdown consumers

	for _, consumer := range plex.consumers {
		consumer.Control() <- shared.ConsumerControlStop
	}
	plex.consumerThreads.Wait()

	// Drain the log channel if there are producers listening

	processLog := len(plex.producers) > 0 && plex.producersStarted
	for processLog {
		select {
		case message := <-shared.Log.Messages:
			plex.broadcastMessage(message)
		default:
			processLog = false
		}
	}

	// Shutdown producers

	for _, producer := range plex.producers {
		producer.Control() <- shared.ProducerControlStop
	}
	plex.producerThreads.Wait()

	// Write remaining messages to stderr

	for {
		select {
		case message := <-shared.Log.Messages:
			fmt.Fprintln(os.Stdout, message.Format(shared.MessageFormatForward))
			message.Data.Release()
		default:
			return
		}
	}
}

// Run the multiplexer.
// Fetch messags from the consumers and pass them to all producers.
func (plex multiplexer) run() {
	defer plex.shutdown()

	if len(plex.consumers) == 0 {
		fmt.Println("Error: No consumers configured.")
		return // ### return, nothing to do ###
	}

	if len(plex.producers) == 0 {
		fmt.Println("Error: No producers configured.")
		return // ### return, nothing to do ###
	}

	// Launch consumers and producers

	for _, producer := range plex.producers {
		go producer.Produce(plex.producerThreads)
	}
	plex.producersStarted = true

	for _, consumer := range plex.consumers {
		go consumer.Consume(plex.consumerThreads)
	}

	// React on signals

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt)

	// Main loop

	shared.Log.Note("We be nice to them, if they be nice to us. (startup)")

	for {
		// Go over all consumers in round-robin fashion
		// Always check for signals
		// Always check for log messages

		for _, consumer := range plex.consumers {
			select {
			case <-signalChannel:
				shared.Log.Note("Master betrayed us. Wicked. Tricksy, False. (signal)")
				return

			case message := <-shared.Log.Messages:
				plex.broadcastMessage(message)

			case message := <-consumer.Messages():
				plex.broadcastMessage(message)
			default:
				// don't block
			}
		}
	}
}
