// Copyright 2015-2019 trivago N.V.
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
	"testing"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tcontainer"
	"github.com/trivago/tgo/ttesting"
)

func TestMultilineGrok(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.GrokToJSON")
	config.Override("Patterns", []string{`(?sm)%{GREEDYDATA:data}`})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*GrokToJSON)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("us-west.servicename.webserver0.this.\nis.\nthe.\nmeasurement 12.0 1497003802"), nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	fmt.Println(msg.String())

	jsonData := tcontainer.NewMarshalMap()
	err = json.Unmarshal(msg.GetPayload(), &jsonData)
	expect.NoError(err)

	expect.MapEqual(jsonData, "data", "us-west.servicename.webserver0.this.\nis.\nthe.\nmeasurement 12.0 1497003802")

	expect.NoError(err)
}

func TestPatternToJSON(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.GrokToJSON")
	config.Override("Patterns", []string{`(?P<datacenter>[^\.]+?)\.(?P<service>[^\.]+?)\.(?P<host>[^\.]+?)\.(?P<measurement>[^\s]+?)\s%{NUMBER:value:float}\s%{INT:time}`})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*GrokToJSON)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("us-west.servicename.webserver0.this.is.the.measurement 12.0 1497003802\r"), nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	jsonData := tcontainer.NewMarshalMap()
	err = json.Unmarshal(msg.GetPayload(), &jsonData)
	expect.NoError(err)

	expect.MapEqual(jsonData, "datacenter", "us-west")
	expect.MapEqual(jsonData, "service", "servicename")
	expect.MapEqual(jsonData, "host", "webserver0")
	expect.MapEqual(jsonData, "measurement", "this.is.the.measurement")
	expect.MapEqual(jsonData, "value", "12.0")
	expect.MapEqual(jsonData, "time", "1497003802")
	expect.NoError(err)
}

func TestMultiplePatternsOrder(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.GrokToJSON")
	config.Override("Patterns", []string{
		`(?P<datacenter>[^\.]+?)\.(?P<service>[^\.]+?)\.(?P<host>[^\.]+?)\.statsd\.gauge-(?P<nomatch>[^\.]+?)\.(?P<measurement>[^\s]+?)\s%{NUMBER:value_gauge:float}\s*%{INT:time}.*`,
		`(?P<datacenter>[^\.]+?)\.(?P<service>[^\.]+?)\.(?P<host>[^\.]+?)\.statsd\.latency-(?P<application>[^\.]+?)\.(?P<measurement>[^\s]+?)\s%{NUMBER:value_latency:float}\s*%{INT:time}.*`,
		`(?P<datacenter>[^\.]+?)\.(?P<service>[^\.]+?)\.(?P<host>[^\.]+?)\.(?P<measurement>[^\s]+?)\s%{NUMBER:value:float}\s*%{INT:time}.*`,
	})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*GrokToJSON)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("us-east.www.webserver1.statsd.latency-appname.another.test-measurement 42.1 1497005802\r"), nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	jsonData := tcontainer.NewMarshalMap()
	err = json.Unmarshal(msg.GetPayload(), &jsonData)
	expect.NoError(err)

	expect.MapEqual(jsonData, "datacenter", "us-east")
	expect.MapEqual(jsonData, "service", "www")
	expect.MapEqual(jsonData, "host", "webserver1")
	expect.MapEqual(jsonData, "application", "appname")
	expect.MapEqual(jsonData, "measurement", "another.test-measurement")
	expect.MapEqual(jsonData, "value_latency", "42.1")
	expect.MapEqual(jsonData, "time", "1497005802")
	expect.NoError(err)
}

func TestNoMatch(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.GrokToJSON")
	config.Override("Patterns", []string{
		`%{INT:random}`,
	})

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*GrokToJSON)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte("nonumber"), nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	if err == nil {
		t.Fatalf("expected error")
	}
}
