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
	"github.com/trivago/gollum/shared"
	"sync"
)

// Null producer plugin
// Configuration example
//
//   - "producer.Null":
//     Enable: true
//
// This producer does nothing.
// It shows the most basic producer who can exist.
// In combination with the consumer "Profiler" this can act as a performance test
type Null struct {
	shared.ProducerBase
}

func init() {
	shared.RuntimeType.Register(Null{})
}

func (prod Null) testFormatter(msg shared.Message) {
	prod.Formatter().PrepareMessage(msg)
	prod.Formatter().String()
}

// Produce writes to a buffer that is dumped to a file.
func (prod Null) Produce(threads *sync.WaitGroup) {
	prod.DefaultControlLoop(prod.testFormatter, nil)
}
