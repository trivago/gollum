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
)

// Double is a formatter that doubles the message and glues both parts
// together by using a separator. Both parts of the new message may be
// formatted differntly
//
//   - "<producer|stream>":
//     Formatter: "format.Double"
//	   Separator: ":"
//     LeftStreamID: false
//     LeftFormatters:
//        - "format.Forward"
//     RightFormatters:
//        - "format.Forward"
//
// Separator sets the separator string placed between both parts.
// This is set to ":" by default.
//
// LeftStreamID uses the stream name result of the left side as the
// streamID of this formatter. Set to false by default.
type Double struct {
	core.FormatterBase
	separator    string
	leftStreamID bool
	left         []core.Formatter
	right        []core.Formatter
}

func init() {
	core.TypeRegistry.Register(Double{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Double) Configure(conf core.PluginConfigReader) error {
	format.FormatterBase.Configure(conf)

	leftPlugins, err := conf.WithError.GetPluginArray("Left", []core.Plugin{})
	if !conf.Errors.Push(err) {
		for _, plugin := range leftPlugins {
			formatter, isFormatter := plugin.(core.Formatter)
			if !isFormatter {
				conf.Errors.Pushf("Plugin is not a valid formatter")
			}
			format.left = append(format.left, formatter)
		}
	}

	rightPlugins, err := conf.WithError.GetPluginArray("Right", []core.Plugin{})
	if !conf.Errors.Push(err) {
		for _, plugin := range rightPlugins {
			formatter, isFormatter := plugin.(core.Formatter)
			if !isFormatter {
				conf.Errors.Pushf("Plugin is not a valid formatter")
			}
			format.right = append(format.right, formatter)
		}
	}

	format.separator = conf.GetString("Separator", ":")
	format.leftStreamID = conf.GetBool("LeftStreamID", false)
	return conf.Errors.OrNil()
}

// Format prepends the Hostname of the message to the message.
func (format *Double) Format(msg core.Message) ([]byte, core.MessageStreamID) {
	leftMsg := msg.Clone()
	for _, formatter := range format.left {
		leftMsg.Data, leftMsg.StreamID = formatter.Format(leftMsg)
	}

	rightMsg := msg.Clone()
	for _, formatter := range format.right {
		rightMsg.Data, rightMsg.StreamID = formatter.Format(rightMsg)
	}

	combined := make([]byte, len(leftMsg.Data)+len(format.separator)+len(rightMsg.Data))
	offset := copy(combined, leftMsg.Data)
	offset += copy(combined[offset:], format.separator)
	offset += copy(combined[offset:], rightMsg.Data)

	if format.leftStreamID {
		return combined, leftMsg.StreamID
	}

	return combined, rightMsg.StreamID
}
