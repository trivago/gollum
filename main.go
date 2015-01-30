package main

import (
	"fmt"
	flag "github.com/docker/docker/pkg/mflag"
	_ "github.com/trivago/gollum/consumer"
	_ "github.com/trivago/gollum/producer"
	"github.com/trivago/gollum/shared"
	"os"
	"runtime/pprof"
)

const (
	gollumMajorVer = 0
	gollumMinorVer = 1
	gollumPatchVer = 0
)

func main() {

	flag.Parse()

	if *versionPtr {
		fmt.Printf("Gollum v%d.%d.%d\n", gollumMajorVer, gollumMinorVer, gollumPatchVer)
		return
	}

	if *helpPtr || *configFilePtr == "" {
		flag.Usage()
		fmt.Println("Nothing to do. We must go.")
		return
	}

	if *cpuProfilePtr != "" {
		file, err := os.Create(*cpuProfilePtr)
		if err != nil {
			panic(err)
		}
		pprof.StartCPUProfile(file)
		defer func() {
			pprof.StopCPUProfile()
			file.Close()
		}()
	}

	stringPool := shared.CreateSlabPool()
	shared.Log.Pool = &stringPool

	// Start the gollum multiplexer

	plex := createMultiplexer(*configFilePtr, &stringPool)
	plex.run()

	// Memory profiling

	if *memProfilePtr != "" {
		file, err := os.Create(*memProfilePtr)
		if err != nil {
			panic(err)
		}
		pprof.WriteHeapProfile(file)
		file.Close()
	}
}
