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
	"bytes"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo"
)

// Trim is a formatter that removes part of the message.
// Configuration example
//
//   - "<producer|stream>":
//     Formatters:
//       - "format.Trim"
//          LeftSeparator: ""
//          RightSeparator: ""
//          LeftOffset: 0
//          RightOffset: 0
type Trim struct {
	core.FormatterBase
	leftSeparator  []byte
	rightSeparator []byte
	leftOffset     int
	rightOffset    int
}

func init() {
	core.TypeRegistry.Register(Trim{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Trim) Configure(conf core.PluginConfig) error {
	errors := tgo.NewErrorStack()
	errors.Push(format.FormatterBase.Configure(conf))

	format.leftSeparator = []byte(errors.String(conf.GetString("LeftSeparator", "")))
	format.rightSeparator = []byte(errors.String(conf.GetString("RightSeparator", "")))
	format.leftOffset = errors.Int(conf.GetInt("LeftOffset", 0))
	format.rightOffset = errors.Int(conf.GetInt("RightOffset", 0))

	return errors.OrNil()
}

// Format removes data from the front and/or back of the message.
func (format *Trim) Format(msg core.Message) ([]byte, core.MessageStreamID) {
	leftOffset := format.leftOffset
	if len(format.leftSeparator) > 0 {
		leftIdx := bytes.Index(msg.Data, format.leftSeparator)
		if leftIdx > 0 {
			leftOffset += leftIdx
		}
	}

	rightOffset := len(msg.Data)
	if len(format.rightSeparator) > 0 {
		rightIdx := bytes.LastIndex(msg.Data, format.rightSeparator)
		if rightIdx > 0 {
			rightOffset = rightIdx
		}
	}
	rightOffset -= format.rightOffset

	return msg.Data[leftOffset:rightOffset], msg.StreamID
}
