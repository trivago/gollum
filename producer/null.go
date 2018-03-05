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
	"github.com/trivago/gollum/core"
	"sync"
)

// Null producer
//
// This producer is meant to be used as a sink for data. It will throw away all
// messages without notice.
//
// Examples
//
//  TrashCan:
//    Type: producer.Null
//    Streams: trash
type Null struct {
	core.DirectProducer `gollumdoc:"embed_type"`
}

func init() {
	core.TypeRegistry.Register(Null{})
}

// Produce starts a control loop only
func (prod *Null) Produce(threads *sync.WaitGroup) {
	prod.MessageControlLoop(func(*core.Message) {})
}
