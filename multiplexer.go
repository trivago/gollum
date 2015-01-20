package main

import (
	"fmt"
	"gollum/shared"
	"os"
	"reflect"
)

type multiplexer struct {
	consumers []shared.Consumer
	producers []shared.Producer
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

	for className, instanceConfigs := range conf.Settings {

		for _, config := range instanceConfigs {
			plugin, pluginType, err := shared.Plugin.Create(className)
			if err != nil {
				panic(err.Error())
			}

			// Register consumer plugins

			if pluginType.Implements(consumerType) {
				typedPlugin := plugin.(shared.Consumer)

				instance, err := typedPlugin.Create(config)
				if err != nil {
					fmt.Println("Error registering ", className, ":", err)
					continue // ### continue ###
				}

				plex.consumers = append(plex.consumers, instance)
				fmt.Println("Added consumer", pluginType)

			}

			// Register producer plugins

			if pluginType.Implements(producerType) {
				typedPlugin := plugin.(shared.Producer)

				instance, err := typedPlugin.Create(config)
				if err != nil {
					fmt.Println("Error registering ", className, ":", err)
					continue // ### continue ###
				}

				plex.producers = append(plex.producers, instance)
				fmt.Println("Added producer", pluginType)
			}
		}
	}

	return plex
}

// Run the multiplexer.
// Fetch messags from the consumers and pass them to all producers.
func (plex multiplexer) run() {

	if len(plex.consumers) == 0 {
		fmt.Println("No consumers configured. Done.")
		return // ### return, nothing to do ###
	}
	if len(plex.producers) == 0 {
		fmt.Println("No producers configured. Done.")
		return // ### return, nothing to do ###
	}

	// Launch plugins

	listeners := make([]reflect.SelectCase, len(plex.consumers))

	for i, consumer := range plex.consumers {
		go consumer.Consume()
		listeners[i] = reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(consumer.Messages())}
	}

	for _, producer := range plex.producers {
		go producer.Produce()
	}

	// Main loop

	for {
		_, value, ok := reflect.Select(listeners)
		if ok {
			for _, producer := range plex.producers {
				producer.Messages() <- value.String()
			}
		}
	}
}
