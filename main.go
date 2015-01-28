package main

import (
	"flag"
	"fmt"
	_ "github.com/trivago/gollum/consumer"
	_ "github.com/trivago/gollum/producer"
	"os"
	"runtime/pprof"
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
	profilePtr := flag.String("cpuprofile", "", "Write profiler results to a given file")

	flag.Parse()

	if *versionPtr {
		fmt.Printf("Gollum v%d.%d.%d\n", gollumMajorVer, gollumMinorVer, gollumPatchVer)
	}

	if *configFilePtr == "" {
		fmt.Println("Nothing to do. We must go.")
		return
	}

	if *profilePtr != "" {
		file, err := os.Create(*profilePtr)
		if err != nil {
			panic(err)
		}
		pprof.StartCPUProfile(file)
		defer func() {
			pprof.StopCPUProfile()
			file.Close()
		}()
	}

	// Start the gollum multiplexer
	plex := createMultiplexer(*configFilePtr)
	plex.run()
}
