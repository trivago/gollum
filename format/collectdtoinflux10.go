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
	"strings"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tio"
	"github.com/trivago/tgo/tmath"
)

// CollectdToInflux10 formatter
//
// This formatter transforms JSON data produced by collectd to InfluxDB 0.9.1 or
// later. Trailing and leading commas are removed from the Collectd message.
//
// Examples
//
//  ExampleConsumer:
//    Type: consumer.Console
//    Streams: console
//    Modulators:
//      - formatter.CollectdToInflux10
type CollectdToInflux10 struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	tagString            *strings.Replacer
}

func init() {
	core.TypeRegistry.Register(CollectdToInflux10{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *CollectdToInflux10) Configure(conf core.PluginConfigReader) {
	format.tagString = strings.NewReplacer(",", "\\,", " ", "\\ ")
}

func (format *CollectdToInflux10) escapeTag(value string) string {
	return format.tagString.Replace(value)
}

// ApplyFormatter update message payload
func (format *CollectdToInflux10) ApplyFormatter(msg *core.Message) error {
	contentData := format.GetAppliedContent(msg)

	collectdData, err := parseCollectdPacket(contentData)
	if err != nil {
		format.Logger.Error("Collectd parser error: ", err)
		return err
	}

	// Manually convert to line protocol
	influxData := tio.NewByteStream(len(contentData))
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

	format.SetAppliedContent(msg, influxData.Bytes())
	return nil
}
