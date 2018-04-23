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

package format

import (
	"strconv"
	"sync/atomic"

	"github.com/trivago/gollum/core"
)

// Sequence formatter
//
// This formatter prefixes data with a sequence number managed by the
// formatter. All messages passing through an instance of the
// formatter will get a unique number. The number is not persisted,
// i.e. it restarts at 0 after each restart of gollum.
//
// Parameters
//
// - Separator: Defines the separator string placed between number and data.
// By default this parameter is set to ":".
//
// Examples
//
// This example will insert the sequence number into an existing JSON payload.
//
//  exampleProducer:
//    Type: producer.Console
//    Streams: "*"
//    Modulators:
//      - format.Trim:
//        LeftSeparator: "{"
//        RightSeparator: "}"
//      - format.Sequence
//        Separator: ","
//      - format.Envelope:
//        Prefix: "{\"seq\":"
//        Postfix: "}"
type Sequence struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	separator            []byte `config:"Separator" default:":"`
	seq                  *int64
}

func init() {
	core.TypeRegistry.Register(Sequence{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Sequence) Configure(conf core.PluginConfigReader) {
	format.seq = new(int64)
}

// ApplyFormatter update message payload
func (format *Sequence) ApplyFormatter(msg *core.Message) error {
	seq := atomic.AddInt64(format.seq, 1)
	sequenceStr := strconv.FormatInt(seq, 10)
	content := format.GetAppliedContent(msg)

	dataSize := len(sequenceStr) + len(format.separator) + len(content)
	payload := make([]byte, dataSize)

	offset := copy(payload, []byte(sequenceStr))
	offset += copy(payload[offset:], format.separator)
	copy(payload[offset:], content)

	format.SetAppliedContent(msg, payload)
	return nil
}
