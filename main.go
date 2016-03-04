// Copyright 2015-2016 trivago GmbH
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
	_ "github.com/trivago/gollum/filter"
	_ "github.com/trivago/gollum/format"
	_ "github.com/trivago/gollum/producer"
	_ "github.com/trivago/gollum/stream"
	"github.com/trivago/tgo"
	"github.com/trivago/tgo/tlog"
	"github.com/trivago/tgo/tstrings"
	"io/ioutil"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
)

const (
	gollumMajorVer = 0
	gollumMinorVer = 5
	gollumPatchVer = 0
	gollumDevVer   = 0
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
	tlog.SetCacheWriter()
	parseFlags()
	tlog.SetVerbosity(tlog.Verbosity(*flagLoglevel))

	contribModules := core.TypeRegistry.GetRegistered("contrib")
	modules := ""
	for _, typeName := range contribModules {
		modules += " + " + typeName[tstrings.IndexN(typeName, ".", 1)+1:] + "\n"
	}

	if *flagVersion {
		if gollumDevVer > 0 {
			fmt.Printf("Gollum v%d.%d.%d.%d dev\n%s", gollumMajorVer, gollumMinorVer, gollumPatchVer, gollumDevVer, modules)
		} else {
			fmt.Printf("Gollum v%d.%d.%d\n%s", gollumMajorVer, gollumMinorVer, gollumPatchVer, modules)
		}
		return // ### return, version only ###
	}

	if *flagHelp || *flagConfigFile == "" {
		printFlags()
		return // ### return, nothing to do ###
	}

	// Read config

	config, err := core.ReadConfig(*flagConfigFile)
	if err != nil {
		fmt.Printf("Config: %s\n", err.Error())
		return // ### return, config error ###
	} else if *flagTestConfigFile {
		fmt.Printf("Config: %s parsed as ok.\n", *flagConfigFile)
		newMultiplexer(config, false)
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

	// Metrics server start

	if *flagMetricsAddress != "" {
		server := tgo.NewMetricServer()
		address := *flagMetricsAddress

		if !strings.Contains(address, ":") {
			if !tstrings.IsInt(address) {
				fmt.Printf("Metrics address must be of the form \"host:port\" or \":port\" or \"port\".\n")
				return
			}
			address = ":" + address
		}

		go server.Start(address)
		defer server.Stop()
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
		defer dumpMemoryProfile()
	}

	// Start the multiplexer

	plex := newMultiplexer(config, *flagProfile)
	plex.run()
}
