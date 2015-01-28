package main

import (
	"fmt"
	"github.com/trivago/gollum/shared"
	"os"
	"os/signal"
	"reflect"
)

type multiplexer struct {
	consumers []shared.Consumer
	producers []shared.Producer
	stream    map[string][]*shared.Producer
}

// Create a new multiplexer based on a given config file.
func createMultiplexer(configFile string) multiplexer {
	conf, err := shared.ReadConfig(configFile)
	if err != nil {
		fmt.Printf("Error: %s", err.Error())
		os.Exit(-1)
	}

	// Initialize the plugins based on the config

	var plex multiplexer
	consumerType := reflect.TypeOf((*shared.Consumer)(nil)).Elem()
	producerType := reflect.TypeOf((*shared.Producer)(nil)).Elem()

	plex.stream = make(map[string][]*shared.Producer)

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

				instance, err := typedPlugin.Create(config)
				if err != nil {
					shared.Log.Error("Failed registering consumer ", className, ":", err)
					continue // ### continue ###
				}

				plex.consumers = append(plex.consumers, instance)
			}

			// Register producer plugins

			if pluginType.Implements(producerType) {
				typedPlugin := plugin.(shared.Producer)

				instance, err := typedPlugin.Create(config)
				if err != nil {
					shared.Log.Error("Failed registering producer ", className, ":", err)
					continue // ### continue ###
				}

				for _, stream := range config.Stream {
					streamMap, exists := plex.stream[stream]
					if !exists {
						streamMap = []*shared.Producer{&instance}
						plex.stream[stream] = streamMap
					} else {
						plex.stream[stream] = append(streamMap, &instance)
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

	if message.Stream != shared.LogInternalStream {
		for _, producer := range plex.stream["*"] {
			if (*producer).Accepts(message) {
				(*producer).Messages() <- message
			}
		}
	}

	// Send to specific stream producers

	for _, producer := range plex.stream[message.Stream] {
		if (*producer).Accepts(message) {
			(*producer).Messages() <- message
		}
	}
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
		<-consumer.ControlResponse()
	}

	// Clear log (we still need a producer to write)
loop:
	for {
		select {
		case message := <-shared.Log.Messages:
			plex.broadcastMessage(message)
		default:
			break loop
		}
	}

	// Shutdown producers

	for _, producer := range plex.producers {
		producer.Control() <- shared.ProducerControlStop
		<-producer.ControlResponse()
	}

	// Write remaining messages to stdout

	for {
		select {
		case message := <-shared.Log.Messages:
			fmt.Println(message.Text)
		default:
			return
		}
	}
}

// Run the multiplexer.
// Fetch messags from the consumers and pass them to all producers.
func (plex multiplexer) run() {

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
		go producer.Produce()
	}

	for _, consumer := range plex.consumers {
		go consumer.Consume()
	}

	// React on signals

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt)

	// Main loop

	defer plex.shutdown()
	shared.Log.Note("We be nice to them, if they be nice to us. (startup)")

	for {
		// Check internal messages

		select {
		case message := <-shared.Log.Messages:
			plex.broadcastMessage(message)
		default:
			// don't block
		}

		// Go over all consumers in round-robin fashion
		// Always check for signals

		for _, consumer := range plex.consumers {
			select {
			case <-signalChannel:
				shared.Log.Note("Master betrayed us. Wicked. Tricksy, False. (signal)")
				return

			case message := <-consumer.Messages():
				plex.broadcastMessage(message)

			default:
				// don't block
			}
		}
	}
}
