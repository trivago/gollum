package main

import (
	"flag"
	"fmt"
	"gollum/shared"
	"os"
	"reflect"
)

type multiplexer struct {
	consumers []shared.Consumer
	producers []shared.Producer
}

func createMultiplexer() multiplexer {
	configFilePtr := flag.String("c", "/etc/gollum.conf", "Configuration file")
	flag.Parse()

	conf, err := shared.ReadConfig(*configFilePtr)
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
			plugin, err := shared.Plugin.Create(className)
			if err != nil {
				panic(err.Error())
			}

			pluginType := reflect.TypeOf(plugin)

			if pluginType.Implements(consumerType) {
				typedPlugin := plugin.(shared.Consumer)

				instance, err := typedPlugin.Create(config)
				if err != nil {
					fmt.Println("Error registering ", className, ":", err)
					continue // ### continue ###
				}

				plex.consumers = append(plex.consumers, instance)
				fmt.Println("Added consumer", pluginType)

			} else if pluginType.Implements(producerType) {
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
