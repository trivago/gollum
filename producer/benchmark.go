// Copyright 2015-2017 trivago GmbH
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
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/shared"
	"sync"
)

// Benchmark producer plugin
// This producer is similar to producer.Null but will use the standard buffer
// mechanism before discarding messages. This producer is used for benchmarking
// the core system. If you require a /dev/null style producer you should
// prefer producer.Null instead as it is way more performant.
// Configuration example
//
//  - "producer.Benchmark":
//
type Benchmark struct {
	core.ProducerBase
}

func init() {
	shared.TypeRegistry.Register(Benchmark{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *Benchmark) Configure(conf core.PluginConfig) error {
	return prod.ProducerBase.Configure(conf)
}

func (prod *Benchmark) discard(msg core.Message) {
}

// Produce discards the message.
func (prod *Benchmark) Produce(workers *sync.WaitGroup) {
	prod.MessageControlLoop(prod.discard)
}
