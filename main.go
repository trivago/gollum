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
	_ "github.com/trivago/gollum/consumer"
	_ "github.com/trivago/gollum/contrib"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/log"
	_ "github.com/trivago/gollum/filter"
	_ "github.com/trivago/gollum/format"
	_ "github.com/trivago/gollum/producer"
	"github.com/trivago/gollum/shared"
	_ "github.com/trivago/gollum/stream"
	"io/ioutil"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"time"
)

const (
	gollumMajorVer = 0
	gollumMinorVer = 2
	gollumPatchVer = 0
)

func dumpMemoryProfile() {
	if file, err := os.Create(*flagMemProfile); err != nil {
		panic(err)
	} else {
		defer file.Close()
		pprof.WriteHeapProfile(file)
	}
}

func main() {
	parseFlags()
	Log.SetVerbosity(Log.Verbosity(*flagLoglevel))

	if *flagVersion {
		fmt.Printf("Gollum v%d.%d.%d\n", gollumMajorVer, gollumMinorVer, gollumPatchVer)
		return // ### return, version only ###
	}

	configFile := flagConfigFile
	if *flagTestConfigFile != "" {
		configFile = flagTestConfigFile
	}

	if *flagHelp || *configFile == "" {
		printFlags()
		return // ### return, nothing to do ###
	}

	// Read config

	config, err := core.ReadConfig(*configFile)
	if err != nil {
		fmt.Printf("Config: %s\n", err.Error())
		return // ### return, config error ###
	} else if *flagTestConfigFile != "" {
		fmt.Printf("Config: %s parsed as ok.\n", *configFile)
		return // ### return, only test config ###
	}

	// Configure runtime

	if *flagPidFile != "" {
		ioutil.WriteFile(*flagPidFile, []byte(strconv.Itoa(os.Getpid())), 0644)
	}

	if *flagNumCPU == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	} else {
		runtime.GOMAXPROCS(*flagNumCPU)
	}

	// Profiling flags

	if *flagCPUProfile != "" {
		if file, err := os.Create(*flagCPUProfile); err != nil {
			panic(err)
		} else {
			defer file.Close()
			pprof.StartCPUProfile(file)
			defer pprof.StopCPUProfile()
		}
	}

	if *flagMemProfile != "" {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		go func() {
			for {
				<-ticker.C
				dumpMemoryProfile()
			}
		}()
	}

	// Metrics server start

	if *flagMetricsPort != 0 {
		server := shared.NewMetricServer()
		go server.Start(*flagMetricsPort)
		defer server.Stop()
	}

	// Start the multiplexer

	plex := newMultiplexer(config, *flagProfile)
	plex.run()
}
