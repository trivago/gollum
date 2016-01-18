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
	"fmt"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/ttesting"
	"testing"
)

func newTestJSONFormatter(directives []interface{}, start string) *JSON {
	format := JSON{}
	conf := core.NewPluginConfig("mockJSONFormatter")
	conf.Typename = "format.JSON"
	conf.Stream = []string{core.LogInternalStream}

	conf.Settings["startstate"] = start
	conf.Settings["directives"] = directives

	if err := format.Configure(conf); err != nil {
		panic(err)
	}
	return &format
}

func TestJSONFormatter1(t *testing.T) {
	expect := ttesting.NewExpect(t)

	testString := `{"a":123,"b":"string","c":[1,2,3],"d":[{"a":1}],"e":[[1,2]],"f":[{"a":1},{"b":2}],"g":[[1,2],[3,4]]}`
	msg := core.NewMessage(nil, []byte(testString), 0)
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

	result, _ := test.Format(msg)
	expect.Equal(testString, string(result))
}

func BenchmarkJSONFormatter(b *testing.B) {

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

	for i := 0; i < b.N; i++ {
		testString := fmt.Sprintf(`{"a":%d23,"b":"string","c":[%d,2,3],"d":[{"a":%d}],"e":[[%d,2]],"f":[{"a":%d},{"b":2}],"g":[[%d,2],[3,4]]}`, i, i, i, i, i, i)
		msg := core.NewMessage(nil, []byte(testString), 0)
		test.Format(msg)
	}
}
