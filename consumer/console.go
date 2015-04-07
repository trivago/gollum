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
//
// This consumer does not define any options beside the standard ones.
type Console struct {
	core.ConsumerBase
}

func init() {
	shared.RuntimeType.Register(Console{})
}

func (cons *Console) readFrom(stream io.Reader) {
	buffer := shared.NewBufferedReader(consoleBufferGrowSize, 0, "\n", cons.Enqueue)

	for {
		err := buffer.Read(stream)
		if err != nil && err != io.EOF {
			Log.Error.Print("Error reading stdin: ", err)
		}
	}
}

// Consume listens to stdin.
func (cons *Console) Consume(workers *sync.WaitGroup) {
	go cons.readFrom(os.Stdin)
	cons.DefaultControlLoop(nil)
}
