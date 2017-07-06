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
	"github.com/trivago/tgo/ttesting"
	"strings"
	"testing"
)

func TestJSONToLineProtocol(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.JSONToInflux10")
	config.Override("Tags", []string{"datacenter", "service", "host", "application"})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*JSONToInflux10)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("{\"time\":\"1320969600\",\"datacenter\":\"us-west\",\"host\":\"webserver0\",\"application\":\"myapp\",\"measurement\":\"users.checkout\",\"value\":\"1000\"}"),
		nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	// The order of tags is arbitrary.
	// Therefore we need to unpack the result for inspection.
	payload := string(msg.GetPayload())
	parts := strings.Fields(payload)
	expect.Equal(len(parts), 3)

	meta := parts[0]
	fields := parts[1]
	timestamp := parts[2]

	expect.Equal(fields, "value=1000")
	expect.Equal(timestamp, "1320969600")

	tags := strings.Split(meta, ",")
	expect.Equal(len(tags), 4)
	measurement := tags[0]
	expect.Equal(measurement, "users.checkout")

	remainingTags := tags[1:]
	declaredTags := make(map[string]bool)
	declaredTags["datacenter=us-west"] = true
	declaredTags["host=webserver0"] = true
	declaredTags["application=myapp"] = true
	for _, t := range remainingTags {
		_, ok := declaredTags[t]
		expect.True(ok)
	}
}

func TestJSONToLineProtocolIgnoreTag(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.JSONToInflux10")
	config.Override("Tags", []string{"foo"})
	config.Override("Ignore", []string{"BASE10NUM"})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*JSONToInflux10)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("{\"BASE10NUM\":\"0.099972\",\"foo\":\"bar\",\"measurement\":\"baz\",\"time\":\"1497606216\",\"value\":\"42\"}"),
		nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	// The order of tags is arbitrary.
	// Therefore we need to unpack the result for inspection.
	payload := string(msg.GetPayload())
	parts := strings.Fields(payload)
	expect.Equal(len(parts), 3)

	meta := parts[0]
	fields := parts[1]
	timestamp := parts[2]

	expect.Equal(fields, "value=42")
	expect.Equal(timestamp, "1497606216")

	tags := strings.Split(meta, ",")
	expect.Equal(len(tags), 2)
	measurement := tags[0]
	expect.Equal(measurement, "baz")
	expect.Equal(tags[1], "foo=bar")
}

func TestJSONToLineProtocolMissingMeasurement(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.JSONToInflux10")
	config.Override("Tags", []string{"datacenter", "service", "host", "application"})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*JSONToInflux10)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("{\"time\":\"1320969600\",\"datacenter\":\"us-west\",\"host\":\"webserver0\",\"application\":\"myapp\",\"something_else\":\"users.checkout\",\"value\":\"1000\"}"),
		nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NotNil(err)
}

func TestJSONToLineProtocolAlternativeMeasurement(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.JSONToInflux10")
	config.Override("Measurement", "hans")
	config.Override("Tags", []string{"missing_tag_is_simply_ignored", "datacenter", "application"})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*JSONToInflux10)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("{\"time\":\"1320969600\",\"datacenter\":\"us-west\",\"value-derive\":\"0.22\",\"application\":\"myapp\",\"hans\":\"users.checkout\",\"value\":\"1000\"}"),
		nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	// The order of tags is arbitrary.
	// Therefore we need to unpack the result for inspection.
	payload := string(msg.GetPayload())
	parts := strings.Fields(payload)
	expect.Equal(len(parts), 3)

	meta := parts[0]
	fields := parts[1]
	timestamp := parts[2]

	remainingFields := strings.Split(fields, ",")
	declaredFields := make(map[string]bool)
	declaredFields["value-derive=0.22"] = true
	declaredFields["value=1000"] = true
	for _, t := range remainingFields {
		_, ok := declaredFields[t]
		expect.True(ok)
	}

	expect.Equal(timestamp, "1320969600")

	tags := strings.Split(meta, ",")
	expect.Equal(len(tags), 3)
	measurement := tags[0]
	expect.Equal(measurement, "users.checkout")

	remainingTags := tags[1:]
	declaredTags := make(map[string]bool)
	declaredTags["datacenter=us-west"] = true
	declaredTags["application=myapp"] = true
	for _, t := range remainingTags {
		_, ok := declaredTags[t]
		expect.True(ok)
	}
}
