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
	"testing"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/ttesting"
)

func TestMultilineGrok(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Grok")
	config.Override("Patterns", []string{`(?sm)%{GREEDYDATA:data}`})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Grok)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("us-west.servicename.webserver0.this.\nis.\nthe.\nmeasurement 12.0 1497003802"), nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	value, err := msg.GetMetadata().String("data")
	expect.NoError(err)

	expect.Equal(value, "us-west.servicename.webserver0.this.\nis.\nthe.\nmeasurement 12.0 1497003802")
}

func TestPatternStoreAtKey(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Grok")
	config.Override("Target", "root")
	config.Override("Patterns", []string{`(?P<datacenter>[^\.]+?)\.(?P<service>[^\.]+?)\.(?P<host>[^\.]+?)\.(?P<measurement>[^\s]+?)\s%{NUMBER:value:float}\s%{INT:time}`})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Grok)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("us-west.servicename.webserver0.this.is.the.measurement 12.0 1497003802\r"), nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expectKey := func(key, val string) {
		v, err := msg.GetMetadata().String(key)
		expect.NoError(err)
		expect.Equal(val, v)
	}

	expectKey("root/datacenter", "us-west")
	expectKey("root/service", "servicename")
	expectKey("root/host", "webserver0")
	expectKey("root/measurement", "this.is.the.measurement")
	expectKey("root/value", "12.0")
	expectKey("root/time", "1497003802")
}

func TestPattern(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Grok")
	config.Override("Patterns", []string{`(?P<datacenter>[^\.]+?)\.(?P<service>[^\.]+?)\.(?P<host>[^\.]+?)\.(?P<measurement>[^\s]+?)\s%{NUMBER:value:float}\s%{INT:time}`})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Grok)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("us-west.servicename.webserver0.this.is.the.measurement 12.0 1497003802\r"), nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expectKey := func(key, val string) {
		v, err := msg.GetMetadata().String(key)
		expect.NoError(err)
		expect.Equal(val, v)
	}

	expectKey("datacenter", "us-west")
	expectKey("service", "servicename")
	expectKey("host", "webserver0")
	expectKey("measurement", "this.is.the.measurement")
	expectKey("value", "12.0")
	expectKey("time", "1497003802")
}

func TestMultiplePatternsOrder(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Grok")
	config.Override("Patterns", []string{
		`(?P<datacenter>[^\.]+?)\.(?P<service>[^\.]+?)\.(?P<host>[^\.]+?)\.statsd\.gauge-(?P<nomatch>[^\.]+?)\.(?P<measurement>[^\s]+?)\s%{NUMBER:value_gauge:float}\s*%{INT:time}.*`,
		`(?P<datacenter>[^\.]+?)\.(?P<service>[^\.]+?)\.(?P<host>[^\.]+?)\.statsd\.latency-(?P<application>[^\.]+?)\.(?P<measurement>[^\s]+?)\s%{NUMBER:value_latency:float}\s*%{INT:time}.*`,
		`(?P<datacenter>[^\.]+?)\.(?P<service>[^\.]+?)\.(?P<host>[^\.]+?)\.(?P<measurement>[^\s]+?)\s%{NUMBER:value:float}\s*%{INT:time}.*`,
	})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Grok)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("us-east.www.webserver1.statsd.latency-appname.another.test-measurement 42.1 1497005802\r"), nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expectKey := func(key, val string) {
		v, err := msg.GetMetadata().String(key)
		expect.NoError(err)
		expect.Equal(val, v)
	}

	expectKey("datacenter", "us-east")
	expectKey("service", "www")
	expectKey("host", "webserver1")
	expectKey("application", "appname")
	expectKey("measurement", "another.test-measurement")
	expectKey("value_latency", "42.1")
	expectKey("time", "1497005802")
}

func TestNoMatch(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.Grok")
	config.Override("Patterns", []string{
		`%{INT:random}`,
	})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*Grok)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("nonumber"), nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NotNil(err)
}
