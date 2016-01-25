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

package consumer

import (
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo"
	"github.com/trivago/tgo/tio"
	"io"
	"os"
	"sync"
)

const (
	consoleBufferGrowSize = 256
)

// Console consumer plugin
// Configuration example
//
//   - "consumer.Console":
//     Enable: true
//	   ExitOnEOF: false
//     Stream:
//       - "console"
//
// This consumer reads from stdin. A message is generated after each newline
// character. When attached to a fuse, this consumer will stop accepting
// messages in case that fuse is burned.
//
// ExitOnEOF can be set to true to trigger an exit signal if StdIn is closed
// (e.g. when a pipe is closed). This is set to false by default.
//
// This consumer does not define any options beside the standard ones.
type Console struct {
	core.ConsumerBase
	autoexit bool
}

func init() {
	core.TypeRegistry.Register(Console{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *Console) Configure(conf core.PluginConfig) error {
	errors := tgo.NewErrorStack()

	cons.autoexit = errors.Bool(conf.GetBool("ExitOnEOF", false))
	err := cons.ConsumerBase.Configure(conf)
	errors.Push(err)

	return errors.OrNil()
}

func (cons *Console) readStdIn() {
	buffer := tio.NewBufferedReader(consoleBufferGrowSize, 0, 0, "\n")

	for cons.IsActive() {
		err := buffer.ReadAll(os.Stdin, cons.Enqueue)
		cons.WaitOnFuse()
		switch err {
		case io.EOF:
			if cons.autoexit {
				// TODO: Hack
				cons.Log.Debug.Print("Console triggered exit.")
				proc, _ := os.FindProcess(os.Getpid())
				proc.Signal(os.Interrupt)
			}

		case nil:
			// ignore
		default:
			cons.Log.Error.Print("Error reading stdin: ", err)
		}
	}
}

// Consume listens to stdin.
func (cons *Console) Consume(workers *sync.WaitGroup) {
	go cons.readStdIn()
	cons.ControlLoop()
}
