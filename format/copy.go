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
	"reflect"
	"strings"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/treflect"
)

// Copy formatter plugin
//
// This formatter sets metadata fields by copying data from the message's
// payload or from other metadata fields.
//
// Parameters
//
// - Source: Defines the key to copy, i.e. the "source" of a copy operation.
// Target will define the target of the copy, i.e. the "destination".
// An empty string will use the message payload as source.
// By default this parameter is set to an empty string (i.e. payload).
//
// - Mode: Defines the copy mode to use. This can be one of "append",
// "prepend" or "replace".
// By default this parameter is set to "replace".
//
// - Separator: When using mode prepend or append, defines the characters
// inserted between source and destination.
// By default this parameter is set to an empty string.
//
// Examples
//
// This example copies the payload to the field key and applies a hash on
// it contain a hash over the complete payload.
//
//  exampleConsumer:
//    Type: consumer.Console
//    Streams: "*"
//    Modulators:
//      - format.Copy:
//        Target: key
//      - formatter.Identifier
//        Generator: hash
//        Target: key
//
type Copy struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	separator            []byte `config:"Separator"`
	mode                 metadataCopyMode
}

type metadataCopyMode int

const (
	metadataCopyModeAppend  = metadataCopyMode(iota)
	metadataCopyModeReplace = metadataCopyMode(iota)
	metadataCopyModePrepend = metadataCopyMode(iota)
)

func init() {
	core.TypeRegistry.Register(Copy{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Copy) Configure(conf core.PluginConfigReader) {
	mode := conf.GetString("Mode", "replace")
	switch strings.ToLower(mode) {
	case "replace":
		format.mode = metadataCopyModeReplace
	case "append":
		format.mode = metadataCopyModeAppend
	case "prepend":
		format.mode = metadataCopyModePrepend
	default:
		conf.Errors.Pushf("mode must be one of replace, append or prepend")
	}
}

func (format *Copy) applyReplace(msg *core.Message) error {
	srcData := format.GetSourceData(msg)
	srcValue := reflect.ValueOf(srcData)

	switch srcValue.Kind() {
	case reflect.Map, reflect.Struct:
		srcData = treflect.Clone(srcValue)

	case reflect.Slice:
		copyValue := reflect.MakeSlice(srcValue.Type(), srcValue.Len(), srcValue.Len())
		reflect.Copy(copyValue, srcValue)
		srcData = copyValue.Interface()
	}

	format.SetTargetData(msg, srcData)

	return nil
}

func (format *Copy) applyAppend(msg *core.Message) error {
	srcData := format.GetSourceDataAsBytes(msg)
	dstData := format.GetTargetDataAsBytes(msg)

	newLen := len(srcData) + len(dstData) + len(format.separator)
	cloneData := make([]byte, len(dstData), newLen)
	copy(cloneData, dstData)
	dstData = cloneData

	if len(format.separator) != 0 {
		dstData = append(dstData, format.separator...)
	}
	format.SetTargetData(msg, append(dstData, srcData...))

	return nil
}

func (format *Copy) applyPrepend(msg *core.Message) error {
	srcData := format.GetSourceDataAsBytes(msg)
	dstData := format.GetTargetDataAsBytes(msg)

	newLen := len(srcData) + len(dstData) + len(format.separator)
	cloneData := make([]byte, len(srcData), newLen)
	copy(cloneData, srcData)
	srcData = cloneData

	if len(format.separator) != 0 {
		srcData = append(srcData, format.separator...)
	}
	format.SetTargetData(msg, append(srcData, dstData...))

	return nil
}

// ApplyFormatter update message payload
func (format *Copy) ApplyFormatter(msg *core.Message) error {
	switch format.mode {
	case metadataCopyModeReplace:
		return format.applyReplace(msg)

	case metadataCopyModePrepend:
		return format.applyPrepend(msg)

	case metadataCopyModeAppend:
		return format.applyAppend(msg)

	default:
		return nil
	}
}
