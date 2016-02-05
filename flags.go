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
	"github.com/trivago/tgo/tflag"
)

var (
	flagHelp           = tflag.Switch("h", "help", "Print this help message.")
	flagVersion        = tflag.Switch("v", "version", "Print version information and quit.")
	flagConfigFile     = tflag.String("c", "config", "", "Use a given configuration file.")
	flagTestConfigFile = tflag.Switch("tc", "testconfig", "Test the given configuration file and exit.")
	flagLoglevel       = tflag.Int("ll", "loglevel", 1, "Set the loglevel [0-3] as in {0=Errors, 1=+Warnings, 2=+Notes, 3=+Debug}.")
	flagNumCPU         = tflag.Int("n", "numcpu", 0, "Number of CPUs to use. Set 0 for all CPUs.")
	flagPidFile        = tflag.String("p", "pidfile", "", "Write the process id into a given file.")
	flagMetricsPort    = tflag.Int("m", "metrics", 0, "Port to use for metric queries. Set 0 to disable.")
	flagCPUProfile     = tflag.String("pc", "profilecpu", "", "Write CPU profiler results to a given file.")
	flagMemProfile     = tflag.String("pm", "profilemem", "", "Write heap profile results to a given file.")
	flagProfile        = tflag.Switch("ps", "profilespeed", "Write msg/sec measurements to log.")
)

func parseFlags() {
	tflag.Parse()
}

func printFlags() {
	tflag.PrintFlags("Usage: gollum [OPTIONS]\n\nGollum - A n:m message multiplexer.\n\nOptions:")
}
