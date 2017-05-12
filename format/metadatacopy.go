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

// MetaDataCopy formatter plugin
// Copy data from defined payload or meta data to set meta data field
//
// Configuration example
//
// - format.MetaDataCopy:
//   WriteTo:
//     - hostname: 	# meta data key
//       Modulators:	# further modulators
//         - format.Hostname
//     - foo:
//       Modulators:
//         - format.Base64Encode
//     - bar 		# 1:1 copy of the payload to meta:bar
//   ApplyTo: "payload" # [payload, meta:key]
//
//
type metaDataMap map[string]core.ModulatorArray

type MetaDataCopy struct {
	core.SimpleFormatter
	metaData          metaDataMap
}

func init() {
	core.TypeRegistry.Register(MetaDataCopy{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *MetaDataCopy) Configure(conf core.PluginConfigReader) error {
	format.SimpleFormatter.Configure(conf)

	format.metaData = format.getMetaDataMapFromArray(conf.GetArray("WriteTo", []interface{}{}))

	return conf.Errors.OrNil()
}

// ApplyFormatter update message payload
func (format *MetaDataCopy) ApplyFormatter(msg *core.Message) error {
	for metaDataKey, modulators := range format.metaData {
		msg.MetaData().SetValue(metaDataKey, format.modulateMetaDataValue(msg, modulators))
	}

	return nil
}

// modulateMetaDataValue returns the final meta value
func (format *MetaDataCopy) modulateMetaDataValue(msg *core.Message, modulators core.ModulatorArray) []byte {
	modulationMsg := core.NewMessage(nil, format.GetAppliedContent(msg), 0, core.InvalidStreamID)

	modulateResult := modulators.Modulate(modulationMsg)
	if modulateResult == core.ModulateResultContinue {
		return modulationMsg.Data()
	}

	return []byte{}
}

// getMetaDataMapFromArray returns a map of meta keys to core.ModulatorArray
func (format *MetaDataCopy) getMetaDataMapFromArray(metaData []interface{}) metaDataMap {
	result := metaDataMap{}
	writeToConfig := core.NewPluginConfig("", "format.MetaDataCopy.WriteTo")

	for _, v := range metaData {
		if converted, isMap := v.(tcontainer.MarshalMap); isMap {
			for keyMetaData, valueMetaData := range converted {

				writeToConfig.Read(valueMetaData.(tcontainer.MarshalMap))
				reader := core.NewPluginConfigReaderWithError(&writeToConfig)

				modulator, err := reader.GetModulatorArray("Modulators", format.Log, core.ModulatorArray{})
				if err != nil {
					format.Log.Error.Print("Can't get mmodulators. Error message: ", err)
					break
				}

				result[keyMetaData] = modulator
			}
		} else if keyMetaData, isString := v.(string); isString {
			// no modulator set => 1:1 copy to meta data key
			result[keyMetaData] = core.ModulatorArray{}
		}
	}

	return result
}
