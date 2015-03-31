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

package format

import (
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/shared"
	"hash/fnv"
	"io"
	"strconv"
	"strings"
)

// Identifier is a formatter that will generate a (mostly) unique 64 bit
// identifier number from the message timestamp and sequence number. The message
// payload will not be encoded.
//
//   - producer.Console
//     Formatter: "format.Identifier"
//     IdentifierType: "hash"
//
// IdentifierType defines the algorithm used to generate the message id.
// This my be one of the following: "hash", "time", "seq", "seqhex".
// By default this is set to "time".
//  * When using "hash" the message payload will be hashed using fnv1a and returned
// as hex.
//  * When using "time" the id will be formatted YYMMDDHHmmSSxxxxxxx where x
// denotes the sequence number modulo 10.000.000. I.e. 10mil messages per second
// are possible before there is a collision.
//  * When using "seq" the id will be returned as the integer representation of
// the sequence number.
//  * When using "seqhex" the id will be returned as the hex representation of
// the sequence number.
type Identifier struct {
	message string
	hash    func(msg core.Message) string
}

func init() {
	shared.RuntimeType.Register(Identifier{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Identifier) Configure(conf core.PluginConfig) error {
	switch strings.ToLower(conf.GetString("IdentifierType", "time")) {
	case "hash":
		format.hash = format.idHash
	case "seq":
		format.hash = format.idSeq
	case "seqhex":
		format.hash = format.idSeqHex
	default:
		fallthrough
	case "time":
		format.hash = format.idTime
	}
	return nil
}

func (format *Identifier) idHash(msg core.Message) string {
	hasher := fnv.New64a()
	hasher.Write(msg.Data)
	return strconv.FormatUint(hasher.Sum64(), 16)
}

func (format *Identifier) idTime(msg core.Message) string {
	return msg.Timestamp.Format("060102150405") + strconv.FormatUint(msg.Sequence%10000000, 10)
}

func (format *Identifier) idSeq(msg core.Message) string {
	return strconv.FormatUint(msg.Sequence, 10)
}

func (format *Identifier) idSeqHex(msg core.Message) string {
	return strconv.FormatUint(msg.Sequence, 16)
}

// PrepareMessage sets the message to be formatted.
func (format *Identifier) PrepareMessage(msg core.Message) {
	format.message = format.hash(msg)
}

// Len returns the length of a formatted message.
func (format *Identifier) Len() int {
	return len(format.message)
}

// String returns the message as string
func (format *Identifier) String() string {
	return string(format.message)
}

// Read copies the message into an existing buffer. It is assumed that
// dest has enough space to fit GetLength() bytes
func (format *Identifier) Read(dest []byte) (int, error) {
	return copy(dest, format.message), nil
}

// WriteTo implements the io.WriterTo interface.
// Data will be written directly to a writer.
func (format *Identifier) WriteTo(writer io.Writer) (int64, error) {
	len, err := writer.Write([]byte(format.message))
	return int64(len), err
}
