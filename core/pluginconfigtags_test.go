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

package core

import (
	"github.com/trivago/tgo/tcontainer"
	"github.com/trivago/tgo/ttesting"
	"net/url"
	"testing"
	"time"
)

type testPluginAutoConfig struct {
	BoolValue      bool              `config:"boolValue"`
	IntValue       int64             `config:"intValue"`
	UintValue      uint64            `config:"uintValue"`
	OctValue       int64             `config:"octValue"`
	DurationValue  time.Duration     `config:"durationValue" metric:"sec"`
	MbValue        int64             `config:"mbValue" metric:"kb"`
	StringValue    string            `config:"stringValue"`
	StringArray    []string          `config:"stringArray"`
	ByteArray      []byte            `config:"byteArray"`
	StreamValue    MessageStreamID   `config:"streamValue"`
	StreamArray    []MessageStreamID `config:"streamArray"`
	Router         Router            `config:"router"`
	RouterArray    []Router          `config:"routerArray"`
	ModulatorArray ModulatorArray    `config:"modulatorArray"`
	FilterArray    FilterArray       `config:"filterArray"`
	FormatterArray FormatterArray    `config:"formatterArray"`
	URLValue       *url.URL          `config:"URLValue"`
	NoURLValue     *url.URL          `config:"NoURLValue"`
}

func (t *testPluginAutoConfig) Configure(conf PluginConfigReader) {
}

func TestConfigReaderAutoConfig(t *testing.T) {
	expect := ttesting.NewExpect(t)
	TypeRegistry.Register(mockPlugin{})
	TypeRegistry.Register(mockFilter{})
	TypeRegistry.Register(mockFormatter{})

	testRouter := getMockRouter()
	StreamRegistry.Register(&testRouter, GetStreamID("test"))
	fooRouter := getMockRouter()
	StreamRegistry.Register(&fooRouter, GetStreamID("foo"))
	barRouter := getMockRouter()
	StreamRegistry.Register(&barRouter, GetStreamID("bar"))

	values := tcontainer.NewMarshalMap()
	values["boolValue"] = true
	values["intValue"] = int64(-1)
	values["uintValue"] = uint64(2)
	values["octValue"] = "017"
	values["durationValue"] = int64(3)
	values["mbValue"] = int64(4)
	values["stringValue"] = "test"
	values["stringArray"] = []string{"foo", "bar"}
	values["byteArray"] = "bytes"
	values["streamValue"] = "test"
	values["streamArray"] = []string{"foo", "bar"}
	values["router"] = "test"
	values["routerArray"] = []string{"foo", "bar"}
	values["filterArray"] = []interface{}{
		map[interface{}]interface{}{"core.mockFilter": tcontainer.NewMarshalMap()},
		map[interface{}]interface{}{"core.mockFilter": tcontainer.NewMarshalMap()},
	}
	values["modulatorArray"] = []interface{}{
		map[interface{}]interface{}{"core.mockFilter": tcontainer.NewMarshalMap()},
		map[interface{}]interface{}{"core.mockFormatter": tcontainer.NewMarshalMap()},
	}
	values["formatterArray"] = []interface{}{
		map[interface{}]interface{}{"core.mockFormatter": tcontainer.NewMarshalMap()},
		map[interface{}]interface{}{"core.mockFormatter": tcontainer.NewMarshalMap()},
	}
	values["URLValue"] = "http://bing.com/search?q=dotnet"
	values["NoURLValue"] = ""

	config, err := NewNestedPluginConfig("core.mockPlugin", values)
	expect.NoError(err)

	reader := NewPluginConfigReader(&config)
	myStruct := testPluginAutoConfig{}
	reader.Configure(&myStruct)

	expect.NoError(reader.Errors.OrNil())
	expect.True(myStruct.BoolValue)
	expect.Equal(int64(-1), myStruct.IntValue)
	expect.Equal(uint64(2), myStruct.UintValue)
	expect.Equal(int64(15), myStruct.OctValue)
	expect.Equal(3*time.Second, myStruct.DurationValue)
	expect.Equal(int64(4096), myStruct.MbValue)
	expect.Equal("test", myStruct.StringValue)
	expect.Equal([]string{"foo", "bar"}, myStruct.StringArray)
	expect.Equal([]byte("bytes"), myStruct.ByteArray)
	expect.Equal(GetStreamID("test"), myStruct.StreamValue)
	expect.Equal([]MessageStreamID{GetStreamID("foo"), GetStreamID("bar")}, myStruct.StreamArray)
	expect.Equal("http://bing.com/search?q=dotnet", myStruct.URLValue.String())
	expect.Nil(myStruct.NoURLValue)

	expect.NotNil(myStruct.Router)
	expect.Equal(&testRouter, myStruct.Router)

	if expect.Equal(2, len(myStruct.RouterArray)) {
		expect.Equal(&fooRouter, myStruct.RouterArray[0])
		expect.Equal(&barRouter, myStruct.RouterArray[1])
	}

	expect.Equal(2, len(myStruct.FilterArray))
	expect.Equal(2, len(myStruct.FormatterArray))
	expect.Equal(2, len(myStruct.ModulatorArray))
}
