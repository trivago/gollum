package main

import (
	"gollum/consumer"
	"gollum/producer"
)

func main() {
	// Call some dummy function so that plugin packages are linked.
	// If we don't do this the plugin packages won't be imported.
	consumer.Initialize()
	producer.Initialize()

	// Start the gollum multiplexer
	plex := createMultiplexer()
	plex.run()
}
