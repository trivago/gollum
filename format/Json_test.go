// Copyright 2015 trivago GmbH
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
	"github.com/trivago/gollum/shared"
	"testing"
)

func newTestJSONFormatter(directives []interface{}, start string) *JSON {
	format := JSON{}
	conf := core.PluginConfig{
		TypeName:  "format.JSON",
		Enable:    true,
		Channel:   1024,
		Instances: 1,
		Stream:    []string{core.LogInternalStream},
		Settings:  make(core.ConfigKeyValueMap),
	}

	conf.Settings["JSONStartState"] = start
	conf.Settings["JSONDirectives"] = directives

	if err := format.Configure(conf); err != nil {
		panic(err)
	}
	return &format
}

func TestJSONFormatter1(t *testing.T) {
	expect := shared.NewExpect(t)

	testString := `{"a":123,"b":"string","c":[1,2,3],"d":[{"a":1}],"e":[[1,2]],"f":[{"a":1},{"b":2}],"g":[[1,2],[3,4]]}`
	msg := core.NewMessage(nil, []byte(testString), []core.MessageStreamID{}, 0)
	test := newTestJSONFormatter([]interface{}{
		`findKey    :":  key        ::`,
		`findKey    :}:             : pop  : end`,
		`key        :":  findVal    :      : key`,
		`findVal    :\:: value      ::`,
		`value      :":  string     ::`,
		`value      :[:  array      : push : arr`,
		`value      :{:  findKey    : push : obj`,
		`value      :,:  findKey    :      : val`,
		`value      :}:             : pop  : val+end`,
		`string     :":  findKey    :      : esc`,
		`array      :[:  array      : push : arr`,
		`array      :{:  findKey    : push : obj`,
		`array      :]:             : pop  : end`,
		`array      :,:  arrIntVal  :      : val`,
		`array      :":  arrStrVal  ::`,
		`arrIntVal  :,:  arrIntVal  :      : val`,
		`arrIntVal  :]:             : pop  : val+end`,
		`arrStrVal  :":  arrNextStr :      : esc`,
		`arrNextStr :":  arrStrVal  ::`,
		`arrNextStr :]:             : pop  : end`,
	}, "findKey")

	test.PrepareMessage(msg)
	expect.StringEq(testString, test.String())
}
