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

package producer

import (
	"fmt"
	"github.com/trivago/gollum/core"
	"os"
	"strings"
	"sync"
)

// Console producer plugin
// Configuration example
//
//   - "producer.Console":
//     Enable: true
//     Console: "stdout"
//
// The console producer writes messages to the standard output streams.
// This producer does not implement a fuse breaker.
//
// Console may either be "stdout" or "stderr". By default it is set to "stdout".
type Console struct {
	core.ProducerBase
	console *os.File
}

func init() {
	core.TypeRegistry.Register(Console{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *Console) Configure(conf core.PluginConfig) error {
	err := prod.ProducerBase.Configure(conf)
	if err != nil {
		return err
	}

	prod.SetStopCallback(prod.close)
	console := conf.GetString("Console", "stdout")

	switch strings.ToLower(console) {
	default:
		fallthrough
	case "stdout":
		prod.console = os.Stdout
	case "stderr":
		prod.console = os.Stderr
	}

	return nil
}

func (prod *Console) printMessage(msg core.Message) {
	text, _ := prod.ProducerBase.Format(msg)
	fmt.Fprint(prod.console, string(text))
}

func (prod *Console) close() {
	prod.CloseMessageChannel(prod.printMessage)
}

// Produce writes to stdout or stderr.
func (prod *Console) Produce(workers *sync.WaitGroup) {
	defer prod.WorkerDone()

	prod.AddMainWorker(workers)
	prod.MessageControlLoop(prod.printMessage)
}
