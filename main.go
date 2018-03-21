// Copyright 2015-2018 trivago N.V.
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
	"io/ioutil"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	_ "github.com/trivago/gollum/consumer"
	"github.com/trivago/gollum/core"
	_ "github.com/trivago/gollum/filter"
	_ "github.com/trivago/gollum/format"
	"github.com/trivago/gollum/logger"
	_ "github.com/trivago/gollum/producer"
	_ "github.com/trivago/gollum/router"
	"github.com/trivago/tgo"
	"github.com/trivago/tgo/thealthcheck"
	"github.com/trivago/tgo/tnet"
	"github.com/trivago/tgo/tos"
	"golang.org/x/crypto/ssh/terminal"
)

// logrusHookBuffer is our single instance of LogrusHookBuffer
var logrusHookBuffer logger.LogrusHookBuffer

func main() {
	exitCode := mainWithExitCode()
	os.Exit(exitCode)
}

func mainWithExitCode() int {
	parseFlags()

	if *flagHelp || len(os.Args) == 1 {
		printFlags()
		return tos.ExitSuccess // ### return, help screen ###
	}

	if *flagExtVersion {
		printExtVersion()
		return tos.ExitSuccess // ### return, version only ###
	}

	if *flagVersion {
		printVersion()
		return tos.ExitSuccess // ### return, version only ###
	}

	if *flagModules {
		printModules()
		return tos.ExitSuccess // ### return, modules only ###
	}

	if stop := initLogrus(); stop != nil {
		defer stop()
	}

	logrus.Debug("GOLLUM STARTING")
	defer logrus.Debug("GOLLUM STOPPED")

	configFile, testConfigAndExit := getConfigFile()
	config := readConfig(configFile)
	if config == nil {
		return tos.ExitError // ### exit, config failed to parse ###
	}

	if testConfigAndExit {
		logrus.SetLevel(logrus.WarnLevel)
		fmt.Println("Testing config", configFile)

		if !testConfig(config) {
			return tos.ExitError // ### exit, config test failed ###
		}

		fmt.Println("Config OK.")
		return tos.ExitSuccess // ### exit, config test successfully ###
	}

	configureRuntime()

	if stop := startMetricsService(); stop != nil {
		defer stop()
	}

	if stop := startHealthCheckService(); stop != nil {
		defer stop()
	}

	if stop := startCPUProfiler(); stop != nil {
		defer stop()
	}

	if stop := startMemoryProfiler(); stop != nil {
		defer stop()
	}

	if stop := startTracer(); stop != nil {
		defer stop()
	}

	coordinator := NewCoordinator()
	defer coordinator.Shutdown()

	if err := coordinator.Configure(config); err != nil {
		logrus.WithError(err).Error("Config validation failed")
		return tos.ExitError // ### exit, config failed to parse ###
	}

	coordinator.StartPlugins()
	coordinator.Run()
	return tos.ExitSuccess
}

func getConfigFile() (configFile string, justTest bool) {
	if *flagTestConfigFile != "" {
		return *flagTestConfigFile, true
	}
	return *flagConfigFile, false
}

// testConfig test and validate config object
func testConfig(config *core.Config) bool {
	coordinator := NewCoordinator()
	defer coordinator.Shutdown()

	if err := coordinator.Configure(config); err != nil {
		logrus.WithError(err).Error("Configure pass failed.")
		return false
	}

	return true
}

// initLogrus initializes the logging framework
func initLogrus() func() {
	// Initialize logger.LogrusHookBuffer
	logrusHookBuffer = logger.NewLogrusHookBuffer()

	// Initialize logging. All logging is done via logrusHookBuffer;
	// logrus's output writer is always set to ioutil.Discard.
	logrus.AddHook(&logrusHookBuffer)
	logrus.SetOutput(ioutil.Discard)
	logrus.SetLevel(getLogrusLevel(*flagLoglevel))

	switch *flagLogColors {
	case "never", "auto", "always":
	default:
		fmt.Printf("Invalid parameter for -log-colors: '%s'\n", *flagLogColors)
		*flagLogColors = "auto"
	}

	if *flagLogColors == "always" ||
		(*flagLogColors == "auto" && terminal.IsTerminal(int(logger.FallbackLogDevice.Fd()))) {
		// Logrus doesn't know the final log device, so we hint the color option here
		logrus.SetFormatter(logger.NewConsoleFormatter())
	}

	// make sure logs are purged at exit
	return func() {
		logrusHookBuffer.SetTargetWriter(logger.FallbackLogDevice)
		logrusHookBuffer.Purge()
	}
}

// readConfig reads and checks the config file for errors.
func readConfig(configFile string) *core.Config {
	if *flagHelp || configFile == "" {
		logrus.Error("Please provide a config file")
		return nil
	}

	config, err := core.ReadConfigFromFile(configFile)
	if err != nil {
		logrus.WithError(err).Error("Failed to read config")
		return nil
	}

	if err := config.Validate(); err != nil {
		logrus.WithError(err).Error("Config validation failed")
		return nil
	}

	return config
}

// configureRuntime does various different settings that affect runtime
// behavior or enables global functionality
func configureRuntime() {
	if *flagPidFile != "" {
		err := ioutil.WriteFile(*flagPidFile, []byte(strconv.Itoa(os.Getpid())), 0644)
		if err != nil {
			logrus.WithError(err).Error("Failed to write pid file")
		}
	}

	if *flagNumCPU == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	} else {
		runtime.GOMAXPROCS(*flagNumCPU)
	}

	if *flagProfile {
		time.AfterFunc(time.Second*3, printProfile)
	}

	if *flagTrace {
		core.ActivateMessageTrace()
	}
}

// startMetricsService creates a metric endpoint if requested.
// The returned function should be deferred if not nil.
func startMetricsService() func() {
	if *flagMetricsAddress == "" {
		return nil
	}

	server := tgo.NewMetricServer()
	address, err := parseAddress(*flagMetricsAddress)
	if err != nil {
		logrus.WithError(err).Error("Failed to start metrics service")
		return nil
	}

	logrus.WithField("address", address).Info("Starting metric service")
	go server.Start(address)
	return server.Stop
}

// startHealthCheckService creates a health check endpoint if requested.
// The returned function should be deferred if not nil.
func startHealthCheckService() func() {
	if *flagHealthCheck == "" {
		return nil
	}
	address, err := parseAddress(*flagHealthCheck)
	if err != nil {
		logrus.WithError(err).Error("Failed to start health check service")
		return nil
	}
	thealthcheck.Configure(address)

	logrus.WithField("address", address).Info("Starting health check service")
	go thealthcheck.Start()

	// Add a static "ping" endpoint
	thealthcheck.AddEndpoint("/_PING_", func() (code int, body string) {
		return thealthcheck.StatusOK, "PONG"
	})
	return thealthcheck.Stop
}

// startCPUProfiler enables the golang CPU profiling process.
// The resulting file can be viewed with `go tool pprof ./gollum file`.
// The returned function should be deferred if not nil.
func startCPUProfiler() func() {
	if *flagCPUProfile == "" {
		return nil
	}

	file, err := os.Create(*flagCPUProfile)
	if err != nil {
		logrus.WithError(err).Error("Failed to create profiling results file")
		return nil
	}

	if err := pprof.StartCPUProfile(file); err != nil {
		file.Close()
		logrus.WithError(err).Error("Failed to start CPU profler")
		return nil
	}

	logrus.WithField("file", *flagCPUProfile).Info("Started CPU profiling")

	return func() {
		pprof.StopCPUProfile()
		file.Close()
	}
}

// startMemoryProfile enables the golang heap profiling process.
// The returned function should be deferred if not nil.
func startMemoryProfiler() func() {
	if *flagMemProfile == "" {
		return nil
	}

	return func() {
		file, err := os.Create(*flagMemProfile)
		if err != nil {
			logrus.WithError(err).Error("Failed to create memory profile results file")
			return
		}
		defer file.Close()

		logrus.WithField("file", *flagMemProfile).Info("Dumping memory profile")
		if err := pprof.WriteHeapProfile(file); err != nil {
			logrus.WithError(err).Error("Failed to write heap profile")
		}
	}
}

// startTracer enables the golang tracing process.
// The resulting file can be viewed with `go tool trace -http=':3333' file` or
// converted to pprof with `go tool trace -pprof=TYPE trace.out > TYPE.pprof`
// where TYPE can be net, sync, syscall or sched.
// The returned function should be deferred if not nil.
func startTracer() func() {
	if *flagProfileTrace == "" {
		return nil
	}

	file, err := os.Create(*flagProfileTrace)
	if err != nil {
		logrus.WithError(err).Error("Failed to create tracing results file")
		return nil
	}

	if err := trace.Start(file); err != nil {
		file.Close()
		logrus.WithError(err).Error("Failed to start tracer")
		return nil
	}

	return func() {
		trace.Stop()
		file.Close()
	}
}

func parseAddress(address string) (string, error) {
	_, host, port, err := tnet.SplitAddress(address, "")
	if err != nil {
		return address, fmt.Errorf("Incorrect address %q: %s", address, err)
	}

	return host + ":" + port, nil
}

func printVersion() {
	fmt.Println(core.GetVersionString())
}

func printExtVersion() {
	fmt.Printf("%6s: %s (%d)\n", "Gollum", core.GetVersionString(), core.GetVersionNumber())
	fmt.Printf("%6s: %s\n", "Go", runtime.Version()[2:])
	fmt.Printf("%6s: %s\n", "Arch", runtime.GOARCH)
}

func printModules() {
	namespaces := []string{"consumer", "producer", "filter", "format", "router", "contrib"}
	allMods := []string{}
	for _, pkg := range namespaces {
		modules := core.TypeRegistry.GetRegistered(pkg)
		allMods = append(allMods, modules...)
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
	msgSec, err := tgo.Metric.Get(core.MetricMessagesRoutedAvg)
	if err == nil {
		fmt.Printf("Processed %d msg/sec\n", msgSec)
	}
	time.AfterFunc(time.Second*3, printProfile)
}
