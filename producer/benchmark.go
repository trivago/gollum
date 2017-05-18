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
	"sync"
)

// Benchmark producer plugin
// The producer is used to benchmark the core system.
type Benchmark struct {
	core.BufferedProducer `gollumdoc:"embed_type"`
}

func init() {
	core.TypeRegistry.Register(Benchmark{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *Benchmark) Configure(conf core.PluginConfigReader) error {
	prod.BufferedProducer.Configure(conf)
	return conf.Errors.OrNil()
}

func (prod *Benchmark) null(msg *core.Message) {
}

// Produce writes to stdout or stderr.
func (prod *Benchmark) Produce(workers *sync.WaitGroup) {
	prod.MessageControlLoop(prod.null)
}
