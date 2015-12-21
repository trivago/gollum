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
	"github.com/trivago/tgo/flag"
)

var (
	flagHelp           = flag.Switch("h", "help", "Print this help message.")
	flagVersion        = flag.Switch("v", "version", "Print version information and quit.")
	flagConfigFile     = flag.String("c", "config", "", "Use a given configuration file.")
	flagTestConfigFile = flag.Switch("tc", "testconfig", "Test the given configuration file and exit.")
	flagLoglevel       = flag.Int("ll", "loglevel", 1, "Set the loglevel [0-3] as in {0=Errors, 1=+Warnings, 2=+Notes, 3=+Debug}.")
	flagNumCPU         = flag.Int("n", "numcpu", 0, "Number of CPUs to use. Set 0 for all CPUs.")
	flagPidFile        = flag.String("p", "pidfile", "", "Write the process id into a given file.")
	flagMetricsPort    = flag.Int("m", "metrics", 0, "Port to use for metric queries. Set 0 to disable.")
	flagCPUProfile     = flag.String("pc", "profilecpu", "", "Write CPU profiler results to a given file.")
	flagMemProfile     = flag.String("pm", "profilemem", "", "Write heap profile results to a given file.")
	flagProfile        = flag.Switch("ps", "profilespeed", "Write msg/sec measurements to log.")
)

func parseFlags() {
	flag.Parse()
}

func printFlags() {
	flag.PrintFlags("Usage: gollum [OPTIONS]\n\nGollum - A n:m message multiplexer.\n\nOptions:")
}
