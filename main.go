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
	"github.com/trivago/gollum/core"
	_ "github.com/trivago/gollum/filter"
	_ "github.com/trivago/gollum/format"
	_ "github.com/trivago/gollum/producer"
	_ "github.com/trivago/gollum/router"
	"github.com/trivago/tgo"
	"github.com/trivago/tgo/thealthcheck"
	"github.com/trivago/tgo/tlog"
	"github.com/trivago/tgo/tstrings"
	"io/ioutil"
	"net"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"sort"
	"strconv"
	"strings"
	"time"
)

func main() {
	tlog.SetCacheWriter()
	parseFlags()
	tlog.SetVerbosity(tlog.Verbosity(*flagLoglevel))

	if *flagVersion {
		printVersion()
		return // ### return, version only ###
	}

	if *flagModules {
		printModules()
		return // ### return, modules only ###
	}

	if *flagHelp || *flagConfigFile == "" {
		printFlags()
		return // ### return, nothing to do ###
	}

	// Read and test config

	configFile := *flagTestConfigFile
	if configFile == "" {
		configFile = *flagConfigFile
	}

	config, err := core.ReadConfig(*flagConfigFile)
	if err != nil {
		fmt.Printf("Config: %s\n", err.Error())
		return // ### return, config error ###
	}

	errors := config.Validate()
	for _, err := range errors {
		fmt.Print(err.Error())
	}

	if *flagTestConfigFile != "" {
		if len(errors) == 0 {
			fmt.Print("Config check passed.")
		} else {
			fmt.Print("Config check FAILED.")
		}
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

	setStaticMetrics()
	if *flagMetricsAddress != "" {
		server := tgo.NewMetricServer()
		address, err := parseAddress(*flagMetricsAddress)
		if err != nil {
			fmt.Printf("%s", err)
			return
		}
		go server.Start(address)
		defer server.Stop()
	}

	// Health Check endpoint

	if *flagHealthCheck != "" {
		address, err := parseAddress(*flagHealthCheck)
		if err != nil {
			fmt.Printf("%s", err)
			return
		}
		thealthcheck.Configure(address)

		go thealthcheck.Start()
		defer thealthcheck.Stop()

		// Add a static "ping" endpoint
		thealthcheck.AddEndpoint("/_PING_", func() (code int, body string) {
			return thealthcheck.StatusOK, "PONG"
		})
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

	if *flagProfile {
		time.AfterFunc(time.Second*3, printProfile)
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

	// Start the coordinator

	coordinater := NewCoordinator()
	coordinater.Configure(config)

	defer coordinater.Shutdown()
	coordinater.StartPlugins()
	coordinater.Run()
}

func parseAddress(address string) (string, error) {
	// net.SplitHostPort() doesn't support plain port number
	if tstrings.IsInt(address) {
		address = ":" + address
	}

	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return address, fmt.Errorf("Incorrect address %q: %s", address, err)
	}

	return host + ":" + port, nil
}

func dumpMemoryProfile() {
	if file, err := os.Create(*flagMemProfile); err != nil {
		panic(err)
	} else {
		defer file.Close()
		pprof.WriteHeapProfile(file)
	}
}

func printVersion() {
	fmt.Printf("Gollum: %s\n", GetVersionString())
	fmt.Printf("Version: %d\n", GetVersionNumber())
	fmt.Println(runtime.Version())
}

func printModules() {
	namespaces := []string{"consumer", "producer", "filter", "format", "router", "contrib"}
	allMods := []string{}
	for _, pkg := range namespaces {
		modules := core.TypeRegistry.GetRegistered(pkg)
		for _, typeName := range modules {
			allMods = append(allMods, typeName)
		}
	}

	sort.Strings(allMods)
	lastCategory := ""

	for _, name := range allMods {
		pkgIdx := strings.LastIndex(name, ".")
		category := name[:pkgIdx]

		if category != lastCategory {
			fmt.Printf("\n-- %s\n", category)
		}

		fmt.Println(name)
		lastCategory = category
	}
}

func printProfile() {
	msgSec, err := tgo.Metric.Get(core.MetricMessagesSec)
	if err == nil {
		fmt.Printf("Processed %d msg/sec\n", msgSec)
	}
	time.AfterFunc(time.Second*3, printProfile)
}

func setStaticMetrics() {
	metricVersion := "Version"
	tgo.Metric.New(metricVersion)
	tgo.Metric.InitSystemMetrics()
	tgo.Metric.Set(metricVersion, GetVersionNumber())
}
