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
	"github.com/trivago/gollum/core"
	"hash/fnv"
	"strconv"
	"strings"
	"sync/atomic"
)

// Identifier formatter
//
// This formatter generates a (mostly) unique 64 bit identifier number from
// the message payload, timestamp and/or sequence number. The number is be
// converted to a human readable form.
//
// Parameters
//
// - Generator: Defines which algorithm to use when generating the identifier.
// This my be one of the following values.
// By default this parameter is set to "time"
//
//  - hash: The message payload will be hashed using fnv1a and returned as hex.
//
//  - time: The id will be formatted YYMMDDHHmmSSxxxxxxx where x denotes the
//  current sequence number modulo 10000000. I.e. 10.000.000 messages per second
//  are possible before a collision occurs.
//
//  - seq: The sequence number will be used.
//
//  - seqhex: The hex encoded sequence number will be used.
//
// Examples
//
// This example will generate a payload checksum and store it to a metadata
// field called "checksum".
//
//  ExampleConsumer:
//    Type: consumer.Console
//    Streams: console
//    Modulators:
//      - formatter.Identifier
//        Generator: hash
//        ApplyTo: checksum
type Identifier struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	hash                 func(*core.Message) []byte
	seq                  *uint64
}

func init() {
	core.TypeRegistry.Register(Identifier{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Identifier) Configure(conf core.PluginConfigReader) {
	idType := strings.ToLower(conf.GetString("Generator", "time"))
	format.seq = new(uint64)

	switch idType {
	case "hash":
		format.hash = format.idHash
	case "seq":
		format.hash = format.idSeq
	case "seqhex":
		format.hash = format.idSeqHex
	case "time":
		format.hash = format.idTime
	default:
		format.hash = func(msg *core.Message) []byte {
			return []byte(idType)
		}
	}
}

func (format *Identifier) idHash(msg *core.Message) []byte {
	hasher := fnv.New64a()
	hasher.Write(msg.GetPayload())
	return []byte(strconv.FormatUint(hasher.Sum64(), 16))
}

func (format *Identifier) idTime(msg *core.Message) []byte {
	seq := atomic.AddUint64(format.seq, 1)
	return []byte(msg.GetCreationTime().Format("060102150405") + strconv.FormatUint(seq%10000000, 10))
}

func (format *Identifier) idSeq(msg *core.Message) []byte {
	seq := atomic.AddUint64(format.seq, 1)
	return []byte(strconv.FormatUint(seq, 10))
}

func (format *Identifier) idSeqHex(msg *core.Message) []byte {
	seq := atomic.AddUint64(format.seq, 1)
	return []byte(strconv.FormatUint(seq, 16))
}

// ApplyFormatter update message payload
func (format *Identifier) ApplyFormatter(msg *core.Message) error {
	format.SetAppliedContent(msg, format.hash(msg))
	return nil
}
