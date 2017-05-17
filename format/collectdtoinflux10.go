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
	"github.com/trivago/tgo/tmath"
	"strings"
)

// CollectdToInflux10 formatter plugin
// CollectdToInflux10 provides a transformation from collectd JSON data to
// InfluxDB 0.9.1+ compatible line protocol data. Trailing and leading commas
// are removed from the Collectd message beforehand.
// Configuration example
//
//  - "stream.Broadcast":
//    Formatter: "format.CollectdToInflux10"
//    CollectdToInflux10Formatter: "format.Forward"
//
// CollectdToInfluxFormatter defines the formatter applied before the conversion
// from Collectd to InfluxDB. By default this is set to format.Forward.
type CollectdToInflux10 struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	tagString    *strings.Replacer
	stringString *strings.Replacer
}

func init() {
	core.TypeRegistry.Register(CollectdToInflux10{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *CollectdToInflux10) Configure(conf core.PluginConfigReader) error {
	format.SimpleFormatter.Configure(conf)

	format.tagString = strings.NewReplacer(",", "\\,", " ", "\\ ")
	format.stringString = strings.NewReplacer("\"", "\\\"")
	return conf.Errors.OrNil()
}

func (format *CollectdToInflux10) escapeTag(value string) string {
	return format.tagString.Replace(value)
}

func (format *CollectdToInflux10) escapeString(value string) string {
	return format.stringString.Replace(value)
}

// ApplyFormatter update message payload
func (format *CollectdToInflux10) ApplyFormatter(msg *core.Message) error {
	collectdData, err := parseCollectdPacket(msg.Data())
	if err != nil {
		format.Log.Error.Print("Collectd parser error: ", err)
		return err
	}

	// Manually convert to line protocol
	influxData := tio.NewByteStream(msg.Len())
	timestamp := int64(collectdData.Time * 1000)
	fixedPart := fmt.Sprintf(
		`%s,plugin_instance=%s,type=%s,type_instance=%s,host=%s`,
		format.escapeTag(collectdData.Plugin),
		format.escapeTag(collectdData.PluginInstance),
		format.escapeTag(collectdData.PluginType),
		format.escapeTag(collectdData.TypeInstance),
		format.escapeTag(collectdData.Host))

	setSize := tmath.Min3I(len(collectdData.Dstypes), len(collectdData.Dsnames), len(collectdData.Values))
	for i := 0; i < setSize; i++ {
		fmt.Fprintf(&influxData,
			`%s,dstype=%s,dsname=%s value=%f %d\n`,
			fixedPart,
			format.escapeTag(collectdData.Dstypes[i]),
			format.escapeTag(collectdData.Dsnames[i]),
			collectdData.Values[i],
			timestamp)
	}

	msg.Store(influxData.Bytes())
	return nil
}
