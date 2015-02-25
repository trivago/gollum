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
