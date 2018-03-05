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

package producer

import (
	"fmt"
	"github.com/trivago/gollum/core"
	"os"
	"strings"
	"sync"
)

// Console producer plugin
//
// The console producer writes messages to standard output or standard error.
//
// Parameters
//
// - Console: Chooses the output device; either "stdout" or "stderr".
// By default this is set to "stdout".
//
// Examples
//
//   StdErrPrinter:
//     Type: producer.Console
//     Streams: myerrorstream
//     Console: stderr
//
type Console struct {
	core.BufferedProducer `gollumdoc:"embed_type"`
	console               *os.File
}

func init() {
	core.TypeRegistry.Register(Console{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *Console) Configure(conf core.PluginConfigReader) {
	console := conf.GetString("Console", "stdout")

	switch strings.ToLower(console) {
	default:
		fallthrough
	case "stdout":
		prod.console = os.Stdout
	case "stderr":
		prod.console = os.Stderr
	}
}

func (prod *Console) printMessage(msg *core.Message) {
	fmt.Fprint(prod.console, msg.String())
}

// Produce writes to stdout or stderr.
func (prod *Console) Produce(workers *sync.WaitGroup) {
	defer prod.WorkerDone()

	prod.AddMainWorker(workers)
	prod.MessageControlLoop(prod.printMessage)
}
