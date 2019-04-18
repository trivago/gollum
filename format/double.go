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
)

// Double formatter plugin
//
// Double is a formatter that duplicates a message and applies two different
// sets of formatters to both sides. After both messages have been processed,
// the value of the field defined as "source" by the double formatter will be
// copied from both copies and merged into the "target" field of the original
// message using a given separator.
//
// Parameters
//
// - Separator: This value sets the separator string placed between both parts.
// This parameter is set to ":" by default.
//
// - UseLeftStreamID: When set to "true", use the stream id of the left side
// (after formatting) as the streamID for the resulting message.
// This parameter is set to "false" by default.
//
// - Left: An optional list of formatters. The first copy of the message (left
// of the delimiter) is passed through these filters.
// This parameter is set to an empty list by default.
//
// - Right: An optional list of formatters. The second copy of the mssage (right
// of the delimiter) is passed through these filters.
// This parameter is set to an empty list by default.
//
// Examples
//
// This example creates a message of the form "<orig>|<hash>", where <orig> is
// the original console input and <hash> its hash.
//
//  exampleConsumer:
//    Type: consumer.Console
//    Streams: "*"
//    Modulators:
//      - format.Double:
//        Separator: "|"
//        Right:
//          - format.Identifier:
//            Generator: hash
type Double struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	separator            []byte              `config:"Separator" default:":"`
	leftStreamID         bool                `config:"UseLeftStreamID" default:"false"`
	left                 core.FormatterArray `config:"Left"`
	right                core.FormatterArray `config:"Right"`
	Target               string
}

func init() {
	core.TypeRegistry.Register(Double{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Double) Configure(conf core.PluginConfigReader) {
	format.Target = conf.GetString("Target", "")
}

// ApplyFormatter update message payload
func (format *Double) ApplyFormatter(msg *core.Message) error {
	leftMsg := msg.Clone()
	rightMsg := msg.Clone()

	// apply sub-formatter
	if err := format.left.ApplyFormatter(leftMsg); err != nil {
		return err
	}

	if err := format.right.ApplyFormatter(rightMsg); err != nil {
		return err
	}

	// update content
	leftData := format.GetSourceDataAsBytes(leftMsg)
	rightData := format.GetSourceDataAsBytes(rightMsg)
	format.SetTargetData(msg, format.mergeData(leftData, rightData))

	// handle streamID
	if format.leftStreamID {
		msg.SetStreamID(leftMsg.GetStreamID())
	} else {
		msg.SetStreamID(rightMsg.GetStreamID())
	}

	// fin
	return nil
}

func (format *Double) mergeData(leftContent []byte, rightContent []byte) []byte {
	size := len(leftContent) + len(format.separator) + len(rightContent)
	content := make([]byte, 0, size)

	content = append(content, leftContent...)
	content = append(content, format.separator...)
	content = append(content, rightContent...)

	return content

}
