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
	"runtime/trace"
	"strconv"
)

const (
	gollumMajorVer = 0
	gollumMinorVer = 4
	gollumPatchVer = 5
	gollumDevVer   = 7
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

	contribModules := shared.TypeRegistry.GetRegistered("contrib")
	modules := ""
	for _, typeName := range contribModules {
		modules += " + " + typeName[shared.IndexN(typeName, ".", 1)+1:] + "\n"
	}

	if *flagReport {
		goVersion := runtime.Version()
		if gollumDevVer > 0 {
			fmt.Printf("Gollum %d.%d.%d.%d dev\n%s\nModules:\n%s", gollumMajorVer, gollumMinorVer, gollumPatchVer, gollumDevVer, goVersion, modules)
		} else {
			fmt.Printf("Gollum %d.%d.%d\n%s\nModules:\n%s", gollumMajorVer, gollumMinorVer, gollumPatchVer, goVersion, modules)
		}
		return // ### return, version only ###
	}

	if *flagVersion {
		if gollumDevVer > 0 {
			fmt.Printf("%d.%d.%d.%d dev\n", gollumMajorVer, gollumMinorVer, gollumPatchVer, gollumDevVer)
		} else {
			fmt.Printf("%d.%d.%d\n", gollumMajorVer, gollumMinorVer, gollumPatchVer)
		}
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

	// Profiling flags

	if *flagCPUProfile != "" {
		if file, err := os.Create(*flagCPUProfile); err != nil {
			panic(err)
		} else {
			defer file.Close()
			if err := pprof.StartCPUProfile(file); err != nil {
				panic(err)
			}
			defer pprof.StopCPUProfile()
		}
	}

	if *flagTrace != "" {
		traceFile, err := os.OpenFile(*flagTrace, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			panic(err)
		}
		defer traceFile.Close()

		if err := trace.Start(traceFile); err != nil {
			panic(err)
		}
		defer trace.Stop()
	}

	if *flagMemProfile != "" {
		defer dumpMemoryProfile()
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
