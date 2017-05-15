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
	"fmt"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tio"
)

// CollectdToInflux08 formatter plugin
// CollectdToInflux08 provides a transformation from collectd JSON data to
// InfluxDB 0.8.x compatible JSON data. Trailing and leading commas are removed
// from the Collectd message beforehand.
// Configuration example
//
//  - "stream.Broadcast":
//    Formatter: "format.CollectdToInflux08"
//    CollectdToInfluxFormatter: "format.Forward"
//
// CollectdToInfluxFormatter defines the formatter applied before the conversion
// from Collectd to InfluxDB. By default this is set to format.Forward.
type CollectdToInflux08 struct {
	core.SimpleFormatter `gollumdoc:embed_type`
}

func init() {
	core.TypeRegistry.Register(CollectdToInflux08{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *CollectdToInflux08) Configure(conf core.PluginConfigReader) error {
	return format.SimpleFormatter.Configure(conf)
}

func (format *CollectdToInflux08) createMetricName(plugin string, pluginInstance string, pluginType string, pluginTypeInstance string, host string) string {
	if pluginInstance != "" {
		pluginInstance = "-" + pluginInstance
	}
	if pluginTypeInstance != "" {
		pluginTypeInstance = "-" + pluginTypeInstance
	}
	return fmt.Sprintf("%s.%s%s.%s%s", host, plugin, pluginInstance, pluginType, pluginTypeInstance)
}

// ApplyFormatter update message payload
func (format *CollectdToInflux08) ApplyFormatter(msg *core.Message) error {
	collectdData, err := parseCollectdPacket(msg.Data())
	if err != nil {
		format.Log.Error.Print("Collectd parser error: ", err)
		return err
	}

	// Manually convert to JSON lines
	influxData := tio.NewByteStream(msg.Len())
	name := format.createMetricName(collectdData.Plugin,
		collectdData.PluginInstance,
		collectdData.PluginType,
		collectdData.TypeInstance,
		collectdData.Host)

	for _, value := range collectdData.Values {
		fmt.Fprintf(&influxData, `{"name": "%s", "columns": ["time", "value"], "points":[[%d, %f]]},`, name, int32(collectdData.Time), value)
	}

	msg.Store(influxData.Bytes())
	return nil
}
