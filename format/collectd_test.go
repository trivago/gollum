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
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/ttesting"
	"testing"
)

const collectdTestPayload = `{"values":[42],"dstypes":["gauge"],"dsnames":["legacy"],"time":1426585562.999,"interval":10.000,"host":"example.com","plugin":"golang","type":"gauge"}`

func TestCollectdToInflux10(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.CollectdToInflux10")
	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*CollectdToInflux10)
	expect.True(casted)

	influxPayload := "golang,plugin_instance=,type=gauge,type_instance=,host=example.com,dstype=gauge,dsname=legacy value=42.000000 1426585562999\\n"

	msg := core.NewMessage(nil, []byte(collectdTestPayload), nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)
	expect.Equal(influxPayload, string(msg.GetPayload()))
}

func TestCollectdToInflux09(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.CollectdToInflux09")
	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*CollectdToInflux09)
	expect.True(casted)

	influxPayload := `{"name": "golang", "timestamp": 1426585562, "precision": "ms", "tags": {"plugin_instance": "", "type": "gauge", "type_instance": "", "host": "example.com", "dstype": "gauge", "dsname": "legacy"}, "fields": {"value": 42.000000} },`

	msg := core.NewMessage(nil, []byte(collectdTestPayload), nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal(influxPayload, string(msg.GetPayload()))
}

func TestCollectdToInflux08(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.CollectdToInflux08")
	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*CollectdToInflux08)
	expect.True(casted)

	influxPayload := `{"name": "example.com.golang.gauge", "columns": ["time", "value"], "points":[[1426585562, 42.000000]]},`

	msg := core.NewMessage(nil, []byte(collectdTestPayload), nil, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal(influxPayload, string(msg.GetPayload()))
}
