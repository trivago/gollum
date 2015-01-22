package main

import (
	"flag"
	"fmt"
	_ "gollum/consumer"
	_ "gollum/producer"
)

const (
	gollumMajorVer = 0
	gollumMinorVer = 1
	gollumPatchVer = 0
)

func main() {
	// Command line parameter parsing
	configFilePtr := flag.String("config", "", "Configuration file")
	versionPtr := flag.Bool("v", false, "Show version and exit")

	flag.Parse()

	if *versionPtr {
		fmt.Printf("Gollum v%d.%d.%d", gollumMajorVer, gollumMinorVer, gollumPatchVer)
	}

	if *configFilePtr == "" {
		fmt.Println("Nothing to do. We must go.")
		return
	}

	// Start the gollum multiplexer
	plex := createMultiplexer(*configFilePtr)
	plex.run()
}
