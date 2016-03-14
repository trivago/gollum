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
	"encoding/json"
	"fmt"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
)

// ExtractJSON formatter plugin
// ExtractJSON is a formatter that extracts a single value from a JSON
// message.
// Configuration example
//
//  - "stream.Broadcast":
//    Formatter: "format.ExtractJSON"
//    ExtractJSONdataFormatter: "format.Forward"
//    ExtractJSONField: ""
//    ExtractJSONTrimField: true
//
// ExtractJSONDataFormatter formatter that will be applied before
// the field is extracted. Set to format.Forward by default.
//
// ExtractJSONField defines the field to extract. This value is empty by
// default. If the field does not exist an empty string is returned.
//
// ExtractJSONTrimField will trim whitspaces from the value if enabled.
// Enabled by default.
type ExtractJSON struct {
	base       core.Formatter
	field      string
	trimValues bool
}

func init() {
	shared.TypeRegistry.Register(ExtractJSON{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *ExtractJSON) Configure(conf core.PluginConfig) error {
	plugin, err := core.NewPluginWithType(conf.GetString("ExtractJSONdataFormatter", "format.Forward"), conf)
	if err != nil {
		return err
	}

	format.base = plugin.(core.Formatter)
	format.field = conf.GetString("ExtractJSONField", "")
	format.trimValues = conf.GetBool("ProcessJSONTrimValues", true)

	return nil
}

// Format modifies the JSON payload of this message
func (format *ExtractJSON) Format(msg core.Message) ([]byte, core.MessageStreamID) {
	data, streamID := format.base.Format(msg)

	values := shared.NewMarshalMap()
	err := json.Unmarshal(data, &values)
	if err != nil {
		Log.Warning.Print("ExtractJSON failed to unmarshal a message: ", err)
		return data, streamID // ### return, malformed data ###
	}

	if value, exists := values[format.field]; exists {
		return []byte(fmt.Sprintf("%v", value)), streamID
	}

	return []byte(""), streamID
}
