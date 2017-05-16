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
	"testing"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/ttesting"
)

func newTestTextToJSONFormatter(directives []interface{}, start string) *TextToJSON {
	format := TextToJSON{}
	conf := core.NewPluginConfig("mockTextToJSONFormatter", "format.TextToJSON")

	conf.Override("startstate", start)
	conf.Override("directives", directives)

	if err := format.Configure(core.NewPluginConfigReader(&conf)); err != nil {
		panic(err)
	}
	return &format
}

func TestTextToJSONFormatter1(t *testing.T) {
	expect := ttesting.NewExpect(t)

	testString := `{"a":123,"b":"string","c":[1,2,3],"d":[{"a":1}],"e":[[1,2]],"f":[{"a":1},{"b":2}],"g":[[1,2],[3,4]]}`
	msg := core.NewMessage(nil, []byte(testString), core.InvalidStreamID)
	test := newTestTextToJSONFormatter([]interface{}{
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
		`array      :]:             : pop  : val+end`,
		`array      :,:  array      :      : val`,
		`array      :":  arrString  ::`,
		`arrString  :":  array      :      : esc`,
	}, "findKey")

	err := test.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal(testString, msg.String())
}

func BenchmarkTextToJSONFormatter(b *testing.B) {

	test := newTestTextToJSONFormatter([]interface{}{
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
		`array      :]:             : pop  : val+end`,
		`array      :,:  array      :      : val`,
		`array      :":  arrString  ::`,
		`arrString  :":  array      :      : esc`,
	}, "findKey")

	for i := 0; i < b.N; i++ {
		testString := fmt.Sprintf(`{"a":%d23,"b":"string","c":[%d,2,3],"d":[{"a":%d}],"e":[[%d,2]],"f":[{"a":%d},{"b":2}],"g":[[%d,2],[3,4]]}`, i, i, i, i, i, i)
		msg := core.NewMessage(nil, []byte(testString), core.InvalidStreamID)
		test.ApplyFormatter(msg)
	}
}

func TestTextToJSONFormatterApplyTo(t *testing.T) {
	expect := ttesting.NewExpect(t)

	format := TextToJSON{}
	directives := []interface{}{
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
		`array      :]:             : pop  : val+end`,
		`array      :,:  array      :      : val`,
		`array      :":  arrString  ::`,
		`arrString  :":  array      :      : esc`,
	}

	conf := core.NewPluginConfig("mockTextToJSONFormatter", "format.TextToJSON")
	conf.Override("startstate", "findKey")
	conf.Override("directives", directives)
	conf.Override("ApplyTo", "foo")

	if err := format.Configure(core.NewPluginConfigReader(&conf)); err != nil {
		panic(err)
	}

	testString := `{"a":123,"b":"string","c":[1,2,3],"d":[{"a":1}],"e":[[1,2]],"f":[{"a":1},{"b":2}],"g":[[1,2],[3,4]]}`
	msg := core.NewMessage(nil, []byte("payload"), core.InvalidStreamID)
	msg.MetaData().SetValue("foo", []byte(testString))


	err := format.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal("payload", msg.String())
	expect.Equal(testString, msg.MetaData().GetValueString("foo"))
}