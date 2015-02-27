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
	"os"
)

var (
	helpPtr       = flag.Bool([]string{"h", "-help"}, false, "Print usage")
	versionPtr    = flag.Bool([]string{"v", "-version"}, false, "Print version information and quit")
	msgProfilePtr = flag.Bool([]string{"t", "-throughput"}, false, "Write msg/sec measurements to log")
	numCPU        = flag.Int([]string{"n", "-numcpu"}, 0, "Number of CPUs to use")
	configFilePtr = flag.String([]string{"c", "-config"}, "", "Configuration file")
	cpuProfilePtr = flag.String([]string{"cp", "-cpuprofile"}, "", "Write cpu profiler results to a given file")
	memProfilePtr = flag.String([]string{"mp", "-memprofile"}, "", "Write heap profile to a given file")
	pidFilePtr    = flag.String([]string{"p", "-pidfile"}, "", "Write the process id into a given file")
)

func init() {
	flag.Usage = func() {
		fmt.Println("Usage: gollum [OPTIONS]\n\nGollum - A n:m message multiplexer.\n\nOptions:")
		flag.CommandLine.SetOutput(os.Stdout)
		flag.PrintDefaults()
		fmt.Print("\n")
	}
}
