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

// TrimToBounds formatter
//
// This formatter searches for separator strings and removes all data left or
// right of this separator.
//
// Parameters
//
// - LeftBounds: The string to search for. Searching starts from the left
// side of the data. If an empty string is given this parameter is ignored.
// By default this parameter is set to "".
//
// - RightBounds: The string to search for. Searching starts from the right
// side of the data. If an empty string is given this parameter is ignored.
// By default this parameter is set to "".
//
// - LeftOffset: Defines the search start index when using LeftBounds.
// By default this parameter is set to 0.
//
// - RightOffset: Defines the search start index when using RightBounds.
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
//      - format.TrimToBounds:
//        LeftBounds: "["
//        RightBounds: "]"
type TrimToBounds struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	leftBounds           []byte `config:"LeftBounds"`
	rightBounds          []byte `config:"RightBounds"`
	leftOffset           int    `config:"LeftOffset" default:"0"`
	rightOffset          int    `config:"RightOffset" default:"0"`
}

func init() {
	core.TypeRegistry.Register(TrimToBounds{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *TrimToBounds) Configure(conf core.PluginConfigReader) {
}

// ApplyFormatter update message payload
func (format *TrimToBounds) ApplyFormatter(msg *core.Message) error {
	content := format.GetSourceDataAsBytes(msg)

	endIdx := len(content) - format.rightOffset
	if endIdx < 0 {
		endIdx = 0
	}

	if endIdx > 0 && len(format.rightBounds) > 0 {
		rightIdx := bytes.LastIndex(content[:endIdx], format.rightBounds)
		if rightIdx >= 0 {
			endIdx = rightIdx
		} else {
			endIdx = len(content)
		}
	}

	startIdx := format.leftOffset
	if startIdx >= len(content) {
		startIdx = len(content) - 1
		if startIdx < 0 {
			startIdx = 0
		}
	}

	if startIdx < len(content) && len(format.leftBounds) > 0 {
		leftIdx := bytes.Index(content[startIdx:], format.leftBounds)
		if leftIdx >= 0 {
			startIdx += leftIdx + 1
		}
	}

	if endIdx < startIdx {
		format.SetTargetData(msg, content[0:0])
	} else {
		format.SetTargetData(msg, content[startIdx:endIdx])
	}
	return nil
}
