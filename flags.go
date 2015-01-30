package main

import (
	"fmt"
	flag "github.com/docker/docker/pkg/mflag"
	"os"
)

var (
	configFilePtr = flag.String([]string{"c", "-config"}, "", "Configuration file")
	versionPtr    = flag.Bool([]string{"v", "-version"}, false, "Print version information and quit")
	helpPtr       = flag.Bool([]string{"h", "-help"}, false, "Print usage")
	profilePtr    = flag.String([]string{"p", "-cpuprofile"}, "", "Write profiler results to a given file")
)

func init() {
	flag.Usage = func() {
		fmt.Fprint(os.Stdout, "Usage: gollum [OPTIONS]\n\nGollum - A n:m message multiplexer.\n\nOptions:\n")

		flag.CommandLine.SetOutput(os.Stdout)

		flag.PrintDefaults()
		fmt.Fprintf(os.Stdout, "\n")
	}
}
