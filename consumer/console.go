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
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
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
// character.
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
	shared.TypeRegistry.Register(Console{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *Console) Configure(conf core.PluginConfig) error {
	cons.autoexit = conf.GetBool("ExitOnEOF", false)
	return cons.ConsumerBase.Configure(conf)
}

func (cons *Console) readStdIn() {
	buffer := shared.NewBufferedReader(consoleBufferGrowSize, 0, 0, "\n")

	for {
		err := buffer.ReadAll(os.Stdin, cons.Enqueue)
		switch err {
		case io.EOF:
			if cons.autoexit {
				proc, _ := os.FindProcess(os.Getpid())
				proc.Signal(os.Interrupt)
			}

		case nil:
			// ignore
		default:
			Log.Error.Print("Error reading stdin: ", err)
		}
	}
}

// Consume listens to stdin.
func (cons *Console) Consume(workers *sync.WaitGroup) {
	go cons.readStdIn()
	cons.DefaultControlLoop()
}
