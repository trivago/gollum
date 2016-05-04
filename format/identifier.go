// Copyright 2015-2016 trivago GmbH
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
)

// Identifier formatter plugin
// Identifier is a formatter that will generate a (mostly) unique 64 bit
// identifier number from the message timestamp and sequence number. The message
// payload will not be encoded.
// Configuration example
//
//  - "stream.Broadcast":
//    Formatter: "format.Identifier"
//    IdentifierType: "hash"
//    IdentifierDataFormatter: "format.Forward"
//
// IdentifierType defines the algorithm used to generate the message id.
// This my be one of the following: "hash", "time", "seq", "seqhex".
// By default this is set to "time".
//  * When using "hash" the message payload will be hashed using fnv1a and returned as hex.
//  * When using "time" the id will be formatted YYMMDDHHmmSSxxxxxxx where x denotes the sequence number modulo 10000000.
//    I.e. 10mil messages per second are possible before there is a collision.
//  * When using "seq" the id will be returned as the integer representation of the sequence number.
//  * When using "seqhex" the id will be returned as the hex representation of the sequence number.
//
// IdentifierDataFormatter defines the formatter for the data that is used to
// build the identifier from. By default this is set to "format.Forward"
type Identifier struct {
	core.SimpleFormatter
	hash func(*core.Message) []byte
}

func init() {
	core.TypeRegistry.Register(Identifier{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Identifier) Configure(conf core.PluginConfigReader) error {
	format.SimpleFormatter.Configure(conf)

	idType := strings.ToLower(conf.GetString("Use", "time"))

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

	return conf.Errors.OrNil()
}

func (format *Identifier) idHash(msg *core.Message) []byte {
	hasher := fnv.New64a()
	hasher.Write(msg.Data())
	return []byte(strconv.FormatUint(hasher.Sum64(), 16))
}

func (format *Identifier) idTime(msg *core.Message) []byte {
	return []byte(msg.Created().Format("060102150405") + strconv.FormatUint(msg.Sequence()%10000000, 10))
}

func (format *Identifier) idSeq(msg *core.Message) []byte {
	return []byte(strconv.FormatUint(msg.Sequence(), 10))
}

func (format *Identifier) idSeqHex(msg *core.Message) []byte {
	return []byte(strconv.FormatUint(msg.Sequence(), 16))
}

// Format generates a unique identifier from the message contents or metadata.
func (format *Identifier) Format(msg *core.Message) {
	msg.Store(format.hash(msg))
}
