package main

import (
	"flag"
	"gollum/consumer"
	"gollum/producer"
)

func main() {
	// Call some dummy function so that plugin packages are linked.
	// If we don't do this the plugin packages won't be imported.
	consumer.Initialize()
	producer.Initialize()

	// Command line parameter parsing
	configFilePtr := flag.String("config", "/etc/gollum.conf", "Configuration file")
	flag.Parse()

	// Start the gollum multiplexer
	plex := createMultiplexer(*configFilePtr)
	plex.run()
}
