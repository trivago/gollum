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
	"fmt"
	"strconv"
	"strings"

	"github.com/trivago/gollum/core"
)

// Cast formatter
//
// This formatter casts a given metadata filed into another type.
//
// - AsType: The type to cast to. Can be either string, bytes, float or int.
// By default this parameter is set to "string".
//
// Examples
//
// This example casts the key "bar" to string.
//
//  exampleConsumer:
//    Type: consumer.Console
//    Streams: stdin
//    Modulators:
//      - format.Cast
//        ApplyTo: bar
//        ToType: "string"
type Cast struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	toType               string `config:"ToType" default:"string"`
	castMessage          func(*core.Message) error
}

func init() {
	core.TypeRegistry.Register(Cast{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Cast) Configure(conf core.PluginConfigReader) {
	format.toType = strings.ToLower(format.toType)
	switch format.toType {
	case "string":
		format.castMessage = format.stringCast
	case "bytes":
		format.castMessage = format.bytesCast
	case "float":
		format.castMessage = format.floatCast
	case "int":
		format.castMessage = format.intCast
	default:
		conf.Errors.Push(fmt.Errorf("Unknown target type for casting"))
		format.castMessage = format.noCast
	}
}

func (format *Cast) stringCast(msg *core.Message) error {
	format.SetTargetData(msg, format.GetSourceDataAsString(msg))
	return nil
}

func (format *Cast) bytesCast(msg *core.Message) error {
	format.SetTargetData(msg, format.GetSourceDataAsBytes(msg))
	return nil
}

func (format *Cast) intCast(msg *core.Message) error {
	intVal, err := strconv.ParseInt(format.GetSourceDataAsString(msg), 10, 64)
	if err != nil {
		return err
	}
	format.SetTargetData(msg, intVal)
	return nil
}

func (format *Cast) floatCast(msg *core.Message) error {
	floatVal, err := strconv.ParseFloat(format.GetSourceDataAsString(msg), 64)
	if err != nil {
		return err
	}
	format.SetTargetData(msg, floatVal)
	return nil
}

func (format *Cast) noCast(msg *core.Message) error {
	return nil
}

// ApplyFormatter update message payload
func (format *Cast) ApplyFormatter(msg *core.Message) error {
	return format.castMessage(msg)
}
