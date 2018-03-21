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

package consumer

import (
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo"
	"github.com/trivago/tgo/tio"
)

const (
	consoleBufferGrowSize = 256
)

// Console consumer
//
// This consumer reads from stdin or a named pipe. A message is generated after
// each newline character.
//
// Metadata
//
// *NOTE: The metadata will only set if the parameter `SetMetadata` is active.*
//
// - pipe: Name of the pipe the message was received on (set)
//
// Parameters
//
// - Pipe: Defines the pipe to read from. This can be "stdin" or the path
// to a named pipe. If the named pipe doesn't exist, it will be created.
// By default this paramater is set to "stdin".
//
// - Permissions: Defines the UNIX filesystem permissions used when creating
// the named pipe as an octal number.
// By default this paramater is set to "0664".
//
// - ExitOnEOF: If set to true, the plusing triggers an exit signal if the
// pipe is closed, i.e. when EOF is detected.
// By default this paramater is set to "true".
//
// - SetMetadata: When this value is set to "true", the fields mentioned in the metadata
// section will be added to each message. Adding metadata will have a
// performance impact on systems with high throughput.
// By default this parameter is set to "false".
//
// Examples
//
// This config reads data from stdin e.g. when starting gollum via unix pipe.
//
//  ConsoleIn:
//    Type: consumer.Console
//    Streams: console
//    Pipe: stdin
type Console struct {
	core.SimpleConsumer `gollumdoc:"embed_type"`
	pipe                *os.File
	pipeName            string `config:"Pipe" default:"stdin"`
	pipePerm            uint32 `config:"Permissions" default:"0644"`
	hasToSetMetadata    bool   `config:"SetMetadata" default:"false"`
	autoExit            bool   `config:"ExitOnEOF" default:"true"`
}

func init() {
	core.TypeRegistry.Register(Console{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *Console) Configure(conf core.PluginConfigReader) {
	switch strings.ToLower(cons.pipeName) {
	case "stdin":
		cons.pipe = os.Stdin
		cons.pipeName = "stdin"
	default:
		cons.pipe = nil
	}
}

// Enqueue creates a new message
func (cons *Console) Enqueue(data []byte) {
	if cons.hasToSetMetadata {
		metaData := core.Metadata{}
		metaData.SetValue("pipe", []byte(cons.pipeName))

		cons.EnqueueWithMetadata(data, metaData)
	} else {
		cons.SimpleConsumer.Enqueue(data)
	}
}

// Consume listens to stdin.
func (cons *Console) Consume(workers *sync.WaitGroup) {
	go cons.readPipe()
	cons.ControlLoop()
}

func (cons *Console) readPipe() {
	if cons.pipe == nil {
		var err error
		if cons.pipe, err = tio.OpenNamedPipe(cons.pipeName, cons.pipePerm); err != nil {
			cons.Logger.Error(err)
			time.AfterFunc(3*time.Second, cons.readPipe)
			return // ### return, try again ###
		}

		defer cons.pipe.Close()
	}

	buffer := tio.NewBufferedReader(consoleBufferGrowSize, 0, 0, "\n")
	for cons.IsActive() {
		err := buffer.ReadAll(cons.pipe, cons.Enqueue)
		switch err {
		case io.EOF:
			if cons.autoExit {
				cons.Logger.Info("Exit triggered by EOF.")
				tgo.ShutdownCallback()
			}

		case nil:
			// ignore
		default:
			cons.Logger.Error(err)
		}
	}
}
