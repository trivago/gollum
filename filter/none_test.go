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

func TestFilterNone(t *testing.T) {
	expect := shared.NewExpect(t)
	conf := core.NewPluginConfig("")

	plugin, err := core.NewPluginWithType("filter.None", conf)
	expect.NoError(err)

	filter, casted := plugin.(*None)
	expect.True(casted)

	msg := core.NewMessage(nil, []byte{}, 0)
	expect.False(filter.Accepts(msg))
}
