package main

import (
	"fmt"
	flag "github.com/docker/docker/pkg/mflag"
	_ "github.com/trivago/gollum/consumer"
	_ "github.com/trivago/gollum/contrib"
	_ "github.com/trivago/gollum/distributor"
	_ "github.com/trivago/gollum/format"
	_ "github.com/trivago/gollum/producer"
	"io/ioutil"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"time"
)

const (
	gollumMajorVer = 0
	gollumMinorVer = 1
	gollumPatchVer = 0
)

func dumpMemoryProfile() {
	if file, err := os.Create(*memProfilePtr); err != nil {
		panic(err)
	} else {
		defer file.Close()
		pprof.WriteHeapProfile(file)
	}
}

func main() {
	flag.Parse()

	if *helpPtr || *configFilePtr == "" {
		flag.Usage()
		fmt.Println("Nothing to do. We must go.")
		return
	}

	if *versionPtr {
		fmt.Printf("Gollum v%d.%d.%d\n", gollumMajorVer, gollumMinorVer, gollumPatchVer)
		return
	}

	if *numCPU == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	} else {
		runtime.GOMAXPROCS(*numCPU)
	}

	if *pidFilePtr != "" {
		ioutil.WriteFile(*pidFilePtr, []byte(strconv.Itoa(os.Getpid())), 0644)
	}

	if *cpuProfilePtr != "" {
		if file, err := os.Create(*cpuProfilePtr); err != nil {
			panic(err)
		} else {
			defer file.Close()
			pprof.StartCPUProfile(file)
			defer pprof.StopCPUProfile()
		}
	}

	if *memProfilePtr != "" {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		go func() {
			for {
				<-ticker.C
				dumpMemoryProfile()
			}
		}()
	}

	// Start the gollum multiplexer

	plex := newMultiplexer(*configFilePtr, *msgProfilePtr)
	plex.run()
}
