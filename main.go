// Copyright 2015 trivago GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	flag "github.com/docker/docker/pkg/mflag"
	_ "github.com/trivago/gollum/consumer"
	_ "github.com/trivago/gollum/contrib"
	_ "github.com/trivago/gollum/distributor"
	_ "github.com/trivago/gollum/format"
	_ "github.com/trivago/gollum/producer"
	"github.com/trivago/gollum/shared"
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

	conf, err := shared.ReadConfig(*configFilePtr)
	if err != nil {
		fmt.Printf("Error: %s", err.Error())
		os.Exit(-1)
	}

	plex := newMultiplexer(conf, *msgProfilePtr)
	plex.run()
}
