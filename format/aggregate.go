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

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tcontainer"
)

// Aggregate formatter plugin
//
// Aggregate is a formatter which can group up further formatter.
// The `Target` settings will be pass on and overwritten in the child formatter.
// This plugin could be useful to setup complex configs with metadata handling in more readable format.
//
// Parameters
//
// - Target: This value chooses the part of the message the formatting
// should be applied to. Use "" to target the message payload; other values
// specify the name of a metadata field to target.
// This value will also used for further child modulators!
// By default this parameter is set to "".
//
// - Modulators: Defines a list of child modulators to be applied to a message when
// it arrives at this formatter. Please try to use only content based formatter and filter!
// If a modulator changes the stream of a message the message is NOT routed to this stream anymore.
//
// Examples
//
// This example show a useful case for format.Aggregate plugin:
//
//  exampleConsumerA:
//    Type: consumer.Console
//    Streams: "foo"
//    Modulators:
//      - format.Aggregate:
//          Target: bar
//          Modulators:
//            - format.Copy
//            - format.Envelope:
//                Postfix: "\n"
//      - format.Aggregate:
//          Target: foo
//          Modulators:
//            - format.Copy
//            - format.Base64Encode
//            - format.Double
//            - format.Envelope:
//                Postfix: "\n"
//
//  # same config as
//  exampleConsumerB:
//    Type: consumer.Console
//    Streams: "bar"
//    Modulators:
//      - format.Copy:
//          Target: foo
//      - format.Copy:
//          Target: foo
//      - format.Base64Encode:
//          Target: foo
//      - format.Double:
//          Target: foo
//      - format.Envelope:
//          Postfix: "\n"
//          Target: foo
//      - format.Envelope:
//          Postfix: "\n"
//          Target: bar
//
type Aggregate struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	modulators           core.ModulatorArray
}

func init() {
	core.TypeRegistry.Register(Aggregate{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Aggregate) Configure(conf core.PluginConfigReader) {
	Target := conf.GetString("Target", "")

	// init modulator array
	modulatorSettings := format.getModulatorSettings(Target, conf)

	config := core.NewPluginConfig("", "format.Aggregate.Modulators")
	config.Override("Modulators", modulatorSettings)
	modulatorReader := core.NewPluginConfigReaderWithError(&config)

	modulatorArray, err := modulatorReader.GetModulatorArray("Modulators", format.Logger, core.ModulatorArray{})
	if err != nil {
		conf.Errors.Push(err)
	}

	format.modulators = modulatorArray
}

// ApplyFormatter execute the formatter
func (format *Aggregate) ApplyFormatter(msg *core.Message) error {
	for _, modulator := range format.modulators {
		modulateResult := modulator.Modulate(msg)
		if modulateResult != core.ModulateResultContinue {
			errMsg := "child modulator discarded or trigger fallback routing. " +
				"Please try to use only contend based formatter and filter as child modulators"
			return fmt.Errorf(errMsg)
		}
	}
	return nil
}

func (format *Aggregate) getModulatorSettings(Target string, conf core.PluginConfigReader) []interface{} {
	finalModulatorMap := []interface{}{}

	for _, childFormatterArray := range conf.GetArray("Modulators", []interface{}{}) {
		childFormatterMap := tcontainer.TryConvertToMarshalMap(childFormatterArray, nil)

		// switch childFormatterMap type to difference between direct modulator- and nested modulator settings.
		switch childFormatter := childFormatterMap.(type) {
		case tcontainer.MarshalMap:
			for childFormatterName, childFormatterItem := range childFormatter {
				childFormatterItemMap := childFormatterItem.(tcontainer.MarshalMap)
				childFormatterItemMap["Target"] = Target

				finalModulatorMap = format.appendModulator(childFormatterName, childFormatterItemMap, finalModulatorMap)

			}
		case string:
			childFormatterItemMap := tcontainer.NewMarshalMap()
			childFormatterItemMap["Target"] = Target

			finalModulatorMap = format.appendModulator(childFormatter, childFormatterItemMap, finalModulatorMap)

		default:
			conf.Errors.Pushf("Malformed modulator settings.")
		}
	}

	return finalModulatorMap
}

func (format *Aggregate) appendModulator(name string, settings tcontainer.MarshalMap, modulatorArray []interface{}) []interface{} {
	modulatorItem := map[string]tcontainer.MarshalMap{}
	modulatorItem[name] = settings
	modulatorArray = append(modulatorArray, modulatorItem)

	return modulatorArray
}
