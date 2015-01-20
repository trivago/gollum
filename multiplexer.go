package main

import (
	"fmt"
	"gollum/shared"
	"os"
	"os/signal"
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

			if !config.Enable {
				continue // ### continue, disabld ###
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
					fmt.Println("Error registering ", className, ":", err)
					continue // ### continue ###
				}

				plex.consumers = append(plex.consumers, instance)
				//fmt.Println("Added consumer", pluginType)

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
				//fmt.Println("Added producer", pluginType)
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

	fmt.Println("We be nice to them, if they be nice to us.")

	// Register signal handler

	listeners := make([]reflect.SelectCase, len(plex.consumers)+1)

	signalChannel := make(chan os.Signal, 1)
	signalType := reflect.TypeOf(os.Interrupt)
	signal.Notify(signalChannel, os.Interrupt)

	listeners[0] = reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(signalChannel)}

	// Launch plugins

	for i, consumer := range plex.consumers {
		go consumer.Consume()
		listeners[i+1] = reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(consumer.Messages())}
	}

	for _, producer := range plex.producers {
		go producer.Produce()
	}

	// Main loop

	for {
		_, value, messageReviecved := reflect.Select(listeners)
		if messageReviecved {

			if reflect.TypeOf(value.Interface()) == signalType {
				fmt.Println("signal recieved")

				for _, consumer := range plex.consumers {
					consumer.Control() <- shared.ConsumerControlStop
					<-consumer.ControlResponse()
				}

				for _, producer := range plex.producers {
					producer.Control() <- shared.ProducerControlStop
					<-producer.ControlResponse()
				}

				break
			}

			message := value.Interface().(shared.Message)

			for _, producer := range plex.producers {
				if producer.Accepts(message) {
					producer.Messages() <- message
				}
			}
		}
	}

	fmt.Println("You're a liar and a thief!")
}
