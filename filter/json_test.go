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

package filter

import (
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/shared"
	"testing"
)

func TestFilterJSON(t *testing.T) {
	expect := shared.NewExpect(t)
	conf := core.NewPluginConfig("")

	conf.Override("FilterReject", map[string]string{"data": "^\\d"})
	conf.Override("FilterAccept", map[string]string{"data": "accept"})
	plugin, err := core.NewPluginWithType("filter.JSON", conf)
	expect.NoError(err)

	filter, casted := plugin.(*JSON)
	expect.True(casted)

	msg1 := core.NewMessage(nil, ([]byte)("{\"data\":\"accept\"}"), 0)
	msg2 := core.NewMessage(nil, ([]byte)("{\"data\":\"0accept\"}"), 0)
	msg3 := core.NewMessage(nil, ([]byte)("{\"data\":\"reject\"}"), 0)

	expect.True(filter.Accepts(msg1))
	expect.False(filter.Accepts(msg2))
	expect.False(filter.Accepts(msg3))

	acceptExp := filter.acceptValues["data"]
	delete(filter.acceptValues, "data")

	expect.True(filter.Accepts(msg1))
	expect.False(filter.Accepts(msg2))
	expect.True(filter.Accepts(msg3))

	delete(filter.rejectValues, "data")

	expect.True(filter.Accepts(msg1))
	expect.True(filter.Accepts(msg2))
	expect.True(filter.Accepts(msg3))

	filter.acceptValues["data"] = acceptExp

	expect.True(filter.Accepts(msg1))
	expect.True(filter.Accepts(msg2))
	expect.False(filter.Accepts(msg3))
}
