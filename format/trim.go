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
	"bytes"

	"github.com/trivago/gollum/core"
)

// Trim formatter
//
// This formatter searches for separator strings and removes all data left or
// right of this separator.
//
// Parameters
//
// - LeftSeparator: The string to search for. Searching starts from the left
// side of the data. If an empty string is given this parameter is ignored.
// By default this parameter is set to "".
//
// - RightSeparator: The string to search for. Searching starts from the right
// side of the data. If an empty string is given this parameter is ignored.
// By default this parameter is set to "".
//
// - LeftOffset: Defines the search start index when using LeftSeparator.
// By default this parameter is set to 0.
//
// - RightOffset: Defines the search start index when using RightSeparator.
// Counting starts from the right side of the message.
// By default this parameter is set to 0.
//
// Examples
//
// This example will reduce data like "foo[bar[foo]bar]foo" to "bar[foo]bar".
//
//  exampleConsumer:
//    Type: consumer.Console
//    Streams: "*"
//    Modulators:
//      - format.Trim:
//        LeftSeparator: "["
//        RightSeparator: "]"
type Trim struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	leftSeparator        []byte `config:"LeftSeparator"`
	rightSeparator       []byte `config:"RightSeparator"`
	leftOffset           int    `config:"LeftOffset" default:"0"`
	rightOffset          int    `config:"RightOffset" default:"0"`
}

func init() {
	core.TypeRegistry.Register(Trim{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Trim) Configure(conf core.PluginConfigReader) {
}

// ApplyFormatter update message payload
func (format *Trim) ApplyFormatter(msg *core.Message) error {
	content := format.GetAppliedContent(msg)
	offset := len(content)

	if len(format.rightSeparator) > 0 {
		rightIdx := bytes.LastIndex(content, format.rightSeparator)
		if rightIdx > 0 {
			offset = rightIdx
		}
	}
	format.extendContent(&content, offset-format.rightOffset)

	offset = format.leftOffset
	if len(format.leftSeparator) > 0 {
		leftIdx := bytes.Index(msg.GetPayload(), format.leftSeparator)
		leftIdx++
		if leftIdx > 0 {
			offset += leftIdx
		}
	}
	content = content[offset:]

	format.SetAppliedContent(msg, content)
	return nil
}

func (format *Trim) extendContent(content *[]byte, size int) {
	switch {
	case size == len(*content):
	case size <= cap(*content):
		*content = (*content)[:size]
	default:
		old := *content
		*content = make([]byte, size)
		copy(*content, old)
	}
}
