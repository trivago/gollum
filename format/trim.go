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
	core.SimpleFormatter
	leftSeparator  []byte
	rightSeparator []byte
	leftOffset     int
	rightOffset    int
}

func init() {
	core.TypeRegistry.Register(Trim{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Trim) Configure(conf core.PluginConfigReader) error {
	format.SimpleFormatter.Configure(conf)

	format.leftSeparator = []byte(conf.GetString("LeftSeparator", ""))
	format.rightSeparator = []byte(conf.GetString("RightSeparator", ""))
	format.leftOffset = conf.GetInt("LeftOffset", 0)
	format.rightOffset = conf.GetInt("RightOffset", 0)

	return conf.Errors.OrNil()
}

// ApplyFormatter update message payload
func (format *Trim) ApplyFormatter(msg *core.Message) error {
	offset := msg.Len()
	if len(format.rightSeparator) > 0 {
		rightIdx := bytes.LastIndex(msg.Data(), format.rightSeparator)
		if rightIdx > 0 {
			offset = rightIdx
		}
	}
	msg.Extend(offset - format.rightOffset)

	offset = format.leftOffset
	if len(format.leftSeparator) > 0 {
		leftIdx := bytes.Index(msg.Data(), format.leftSeparator)
		leftIdx += 1
		if leftIdx > 0 {
			offset += leftIdx
		}
	}
	msg.Offset(offset)
	return nil
}