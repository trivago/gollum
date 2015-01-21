package main

import (
	"flag"
	"fmt"
	"gollum/consumer"
	"gollum/producer"
)

func main() {
	// Call some dummy function so that plugin packages are linked.
	// If we don't do this the plugin packages won't be imported.
	consumer.Initialize()
	producer.Initialize()

	// Command line parameter parsing
	configFilePtr := flag.String("config", "", "Configuration file")
	versionPtr := flag.Bool("v", false, "Show version and exit")

	flag.Parse()

	if *versionPtr {
		fmt.Println("Gollum v0.0.0")
	}

	if *configFilePtr == "" {
		fmt.Println("Nothing to do. We must go.")
		return
	}

	// Start the gollum multiplexer
	plex := createMultiplexer(*configFilePtr)
	plex.run()
}
