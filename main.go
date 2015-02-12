package main

import (
	"fmt"
	flag "github.com/docker/docker/pkg/mflag"
	_ "github.com/trivago/gollum/consumer"
	_ "github.com/trivago/gollum/format"
	_ "github.com/trivago/gollum/producer"
	_ "github.com/trivago/gollum/trivago"
	"io/ioutil"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
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

	if *numCPU == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	} else {
		runtime.GOMAXPROCS(*numCPU)
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

	if *pidFilePtr != "" {
		ioutil.WriteFile(*pidFilePtr, []byte(strconv.Itoa(os.Getpid())), 0644)
	}
	// Start the gollum multiplexer

	plex := newMultiplexer(*configFilePtr, *msgProfilePtr)
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
