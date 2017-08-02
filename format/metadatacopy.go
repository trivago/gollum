// Copyright 2015-2017 trivago GmbH
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
	"github.com/trivago/tgo/tcontainer"
)

type metaDataMap map[string]core.ModulatorArray

// MetadataCopy formatter plugin
//
// This formatter sets metadata fields by copying data from the message's
// payload or from other metadata fields.
//
// Parameters
//
// - WriteTo: A named list of meta data keys. Each entry can contain further modulators
// to change or filter the message content before setting the meta data value.
//
// Examples
//
// This example sets the meta fields `hostname`, `base64Value` and `foo` of each message:
//
//  exampleConsumer:
//    Type: consumer.Console
//    Streams: "*"
//    Modulators:
//      - format.MetadataCopy:
//        WriteTo:
//          - hostname:             # meta data key
//            - format.Hostname     # further modulators
//          - base64Value:
//            - format.Base64Encode
//          - payloadCopy           # 1:1 copy of the "payload" to "bar"
type MetadataCopy struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	metaData             metaDataMap
}

func init() {
	core.TypeRegistry.Register(MetadataCopy{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *MetadataCopy) Configure(conf core.PluginConfigReader) {
	format.metaData = format.getMetadataMapFromArray(conf.GetArray("WriteTo", []interface{}{}))
}

// ApplyFormatter update message payload
func (format *MetadataCopy) ApplyFormatter(msg *core.Message) error {
	for metaDataKey, modulators := range format.metaData {
		msg.GetMetadata().SetValue(metaDataKey, format.modulateMetadataValue(msg, modulators))
	}

	return nil
}

// modulateMetadataValue returns the final meta value
func (format *MetadataCopy) modulateMetadataValue(msg *core.Message, modulators core.ModulatorArray) []byte {
	modulationMsg := core.NewMessage(nil, format.GetAppliedContent(msg), nil, core.InvalidStreamID)

	modulateResult := modulators.Modulate(modulationMsg)
	if modulateResult == core.ModulateResultContinue {
		return modulationMsg.GetPayload()
	}

	return []byte{}
}

// getMetadataMapFromArray returns a map of meta keys to core.ModulatorArray
func (format *MetadataCopy) getMetadataMapFromArray(metaData []interface{}) metaDataMap {
	result := metaDataMap{}
	writeToConfig := core.NewPluginConfig("", "format.MetadataCopy.WriteTo")

	for _, metaDataValue := range metaData {
		if converted, isMap := metaDataValue.(tcontainer.MarshalMap); isMap {
			writeToConfig.Read(converted)
			reader := core.NewPluginConfigReaderWithError(&writeToConfig)

			for keyMetadata := range converted {

				modulator, err := reader.GetModulatorArray(keyMetadata, format.Logger, core.ModulatorArray{})
				if err != nil {
					format.Logger.Error("Can't get mmodulators. Error message: ", err)
					break
				}

				result[keyMetadata] = modulator
			}
		} else if keyMetadata, isString := metaDataValue.(string); isString {
			// no modulator set => 1:1 copy to meta data key
			result[keyMetadata] = core.ModulatorArray{}
		}
	}

	return result
}
