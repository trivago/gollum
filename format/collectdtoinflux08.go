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
	"github.com/trivago/tgo/tio"
)

// CollectdToInflux08 formatter
//
// This formatter transforms JSON data produced by collectd to InfluxDB 0.8.x.
// Trailing and leading commas are removed from the Collectd message.
//
// Examples
//
//  ExampleConsumer:
//    Type: consumer.Console
//    Streams: console
//    Modulators:
//      - formatter.CollectdToInflux08
type CollectdToInflux08 struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
}

func init() {
	core.TypeRegistry.Register(CollectdToInflux08{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *CollectdToInflux08) Configure(conf core.PluginConfigReader) {
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
	contentData := format.GetAppliedContent(msg)

	collectdData, err := parseCollectdPacket(contentData)
	if err != nil {
		format.Logger.Error("Collectd parser error: ", err)
		return err
	}

	// Manually convert to JSON lines
	influxData := tio.NewByteStream(len(contentData))
	name := format.createMetricName(collectdData.Plugin,
		collectdData.PluginInstance,
		collectdData.PluginType,
		collectdData.TypeInstance,
		collectdData.Host)

	for _, value := range collectdData.Values {
		fmt.Fprintf(&influxData, `{"name": "%s", "columns": ["time", "value"], "points":[[%d, %f]]},`, name, int32(collectdData.Time), value)
	}

	format.SetAppliedContent(msg, influxData.Bytes())
	return nil
}
