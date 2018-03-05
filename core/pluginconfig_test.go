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
	"testing"
)

// Function checks if non-predefined exists and has been accessed or not
// Plan:
//  Create a new PluginConfig
//  Access few keys
//  check True for existing & accessed keys and false otherwise
//
func TestPluginConfigValidate(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockPluginCfg := NewPluginConfig("", "core.mockPlugin")
	mockPluginCfgReader := NewPluginConfigReaderWithError(&mockPluginCfg)

	// Note that thes values have to be lowercase
	mockPluginCfg.Override("stringKey", "value")
	mockPluginCfg.Override("number", 1)

	// access one field
	sValue, err := mockPluginCfgReader.GetString("stringKey", "")
	expect.NoError(err)
	expect.Equal(sValue, "value")
	err = mockPluginCfg.Validate()
	expect.NoError(err)

	// access second one
	iValue, err := mockPluginCfgReader.GetInt("number", 0)
	expect.NoError(err)
	expect.Equal(iValue, int64(1))
	err = mockPluginCfg.Validate()
	expect.NoError(err)
}

// Function reads initializes pluginConfig with predefined values and
// non-predefined values in the Settings
// Plan:
//  Create a new PluginConfig
//  Create a new tgo.MarshalMap with mock key values
//  Check if they are actually added, if they are, assert the key-value correctness
func TestPluginConfigRead(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockPluginCfg := NewPluginConfig("", "core.mockPlugin")
	mockPluginCfgReader := NewPluginConfigReaderWithError(&mockPluginCfg)

	//reset mockPluginCfg
	mockPluginCfg = NewPluginConfig("", "core.mockPlugin")
	mockPluginCfgReader = NewPluginConfigReaderWithError(&mockPluginCfg)

	testMarshalMap := tcontainer.NewMarshalMap()
	testMarshalMap["Enable"] = true
	testMarshalMap["Host"] = "someHost"
	testMarshalMap["Database"] = "someDatabase"

	mockPluginCfg.Read(testMarshalMap)

	// Check for the bundled config options
	expect.True(mockPluginCfg.Enable)

	// check for the miscelleneous settings key added
	host, err := mockPluginCfgReader.GetString("Host", "")
	expect.NoError(err)
	expect.Equal(host, "someHost")

	db, err := mockPluginCfgReader.GetString("Database", "")
	expect.NoError(err)
	expect.Equal(db, "someDatabase")
}

// Function checks if predefined value exists or not
// Plan:
//  Create a new PluginConfig
//  Add non-predefined Values
//  Check if added ones return true, and others false
func TestPluginConfigHasValue(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockPluginCfg := NewPluginConfig("", "mockPlugin")
	mockPluginCfgReader := NewPluginConfigReaderWithError(&mockPluginCfg)

	expect.False(mockPluginCfgReader.HasValue("aKey"))

	mockPluginCfg.Override("aKey", 1)
	expect.True(mockPluginCfgReader.HasValue("aKey"))
}

// Function sets or overrides value for non-predefined options
// Plan:
//  Create a new PluginConfig
//  Add a new key and a value^
//  Check if it exists. Should be registered
//  Assert key value is correct
//  Override the value to something else
//  Assert new value took effect
func TestPluginConfigOverride(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockPluginCfg := NewPluginConfig("", "core.mockPlugin")
	mockPluginCfgReader := NewPluginConfigReaderWithError(&mockPluginCfg)

	mockPluginCfg.Override("akey", "aValue")
	// make sure the value exists
	expect.True(mockPluginCfgReader.HasValue("akey"))

	// TODO: maybe check for the validate here. Since the value is already checked

	value, err := mockPluginCfgReader.GetString("akey", "")
	expect.NoError(err)
	expect.Equal(value, "aValue")

	// override
	mockPluginCfg.Override("akey", "newValue")

	//check for new value
	value, err = mockPluginCfgReader.GetString("akey", "")
	expect.NoError(err)
	expect.Equal(value, "newValue")

	// TODO: another test with marshalMap
}

// Function gets the string value for a key or if the key/value(?) doesn't
// exist, returns the default value
// Plan:
//  Create a new PluginConfig
//  For a random key, check if the default value is returned
//  Add a key with a value
//  Asserts the string returned for the key is correct
func TestPluginConfigGetString(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockPluginCfg := NewPluginConfig("", "core.mockPlugin")
	mockPluginCfgReader := NewPluginConfigReaderWithError(&mockPluginCfg)

	//check for non-existent key
	value, err := mockPluginCfgReader.GetString("aKey", "default")
	expect.NoError(err)
	expect.Equal(value, "default")

	mockPluginCfg.Override("aKey", "aValue")
	value, err = mockPluginCfgReader.GetString("aKey", "default")
	expect.NoError(err)
	expect.Equal(value, "aValue")
}

// Function gets the string array for a key or default value if non existent
// Plan: Similar to TestPluginConfigGetString
func TestPluginConfigGetStringArray(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockPluginCfg := NewPluginConfig("", "core.mockPlugin")
	mockPluginCfgReader := NewPluginConfigReaderWithError(&mockPluginCfg)

	mockStringArray := []string{"el1", "el2", "el3"}

	// default value is returned
	value, err := mockPluginCfgReader.GetStringArray("arrKey", []string{})
	expect.NoError(err)
	expect.Equal(len(value), 0)

	mockPluginCfg.Override("arrKey", mockStringArray)
	//since expect.Equal is doing reflect.deepValueEqual, arrays should be properly compared
	value, err = mockPluginCfgReader.GetStringArray("arrKey", []string{})
	expect.NoError(err)
	expect.Equal(value, mockStringArray)

}

// Function gets the stringMap for a key or default value if not existent
// Plan: Similar to TestPluginConfigGetString but the Map structure needs assertion
func TestPluginConfigGetStringMap(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockPluginCfg := NewPluginConfig("", "core.mockPlugin")
	mockPluginCfgReader := NewPluginConfigReaderWithError(&mockPluginCfg)

	mockStringMap := map[string]string{
		"k1": "v1",
		"k2": "v2",
		"k3": "v3",
	}

	value, err := mockPluginCfgReader.GetStringMap("strmapkey", map[string]string{})
	expect.NoError(err)
	expect.Equal(len(value), 0)

	mockPluginCfg.Override("strmapkey", mockStringMap)

	value, err = mockPluginCfgReader.GetStringMap("strmapkey", map[string]string{})
	expect.NoError(err)
	expect.Equal(value, mockStringMap)
}

// Function gets an array of MessageStreamID for a given key
// Plan:
//  Create a new PluginConfig
//  Add an array of streams with mocked values in Settings
//  get the streamArray
//  Check the hash received hash with manual generation from streamregistery.getStreamID
func TestPluginConfigGetStreamArray(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockPluginCfg := NewPluginConfig("", "core.mockPlugin")
	mockPluginCfgReader := NewPluginConfigReaderWithError(&mockPluginCfg)

	mockStreamArray := []string{
		"stream1",
		"stream2",
	}

	mockStreamHashed := []MessageStreamID{
		StreamRegistry.GetStreamID("stream1"),
		StreamRegistry.GetStreamID("stream2"),
	}

	value, err := mockPluginCfgReader.GetStreamArray("mockstream", []MessageStreamID{})
	expect.NoError(err)
	expect.Equal(len(value), 0)

	mockPluginCfg.Override("mockstream", mockStreamArray)

	value, err = mockPluginCfgReader.GetStreamArray("mockstream", []MessageStreamID{})
	expect.NoError(err)
	expect.Equal(value, mockStreamHashed)
}

// Function gets a streamMap where key is a streamID and value is a a string
// Plan:
//  Create a new PluginConfig
//  add a new StreamMap in the settings
//  get the streamMap with defaultValue
//  verifiy the hash values
//  get the streamMap without defaultValue
//  verify the hash values
func TestPluginConfigGetStreamMap(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockPluginCfg := NewPluginConfig("", "core.mockPlugin")
	mockPluginCfgReader := NewPluginConfigReaderWithError(&mockPluginCfg)
	defaultValue := "v0"

	mockStringMap := map[string]string{
		"k1": "v1",
		"k2": "v2",
		"k3": "v3",
	}

	expectedMapWithWildcard := map[MessageStreamID]string{
		WildcardStreamID:                 defaultValue,
		StreamRegistry.GetStreamID("k1"): "v1",
		StreamRegistry.GetStreamID("k2"): "v2",
		StreamRegistry.GetStreamID("k3"): "v3",
	}

	expectedMapWithoutWildcard := map[MessageStreamID]string{
		StreamRegistry.GetStreamID("k1"): "v1",
		StreamRegistry.GetStreamID("k2"): "v2",
		StreamRegistry.GetStreamID("k3"): "v3",
	}

	expectedMapOnlyWildCard := map[MessageStreamID]string{
		WildcardStreamID: defaultValue,
	}

	// should be empty because default value is empty, wildcard returned and the key doesn't exist
	value, err := mockPluginCfgReader.GetStreamMap("streammap", "")
	expect.NoError(err)
	expect.Equal(value, map[MessageStreamID]string{})

	// should return wildcard stream when default value not empty and the key doesn't exist
	value, err = mockPluginCfgReader.GetStreamMap("streammap", defaultValue)
	expect.NoError(err)
	expect.Equal(value, expectedMapOnlyWildCard)

	mockPluginCfg.Override("streammap", mockStringMap)
	// without default value, hashed map without wildcard should be returned
	value, err = mockPluginCfgReader.GetStreamMap("streammap", "")
	expect.NoError(err)
	expect.Equal(value, expectedMapWithoutWildcard)

	// with default value, hashed map with wildcard should be returned
	value, err = mockPluginCfgReader.GetStreamMap("streammap", defaultValue)
	expect.NoError(err)
	expect.Equal(value, expectedMapWithWildcard)

}

// Function gets streamRoutes where key is a string and value is a streamID
// Plan: similar to TestPluginConfigGetStreamMap with streamId and value swapped
func TestPluginConfigGetStreamRoutes(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockPluginCfg := NewPluginConfig("", "core.mockPlugin")
	mockPluginCfgReader := NewPluginConfigReaderWithError(&mockPluginCfg)

	mockStreamRoute := map[string][]string{
		"k1": {"v1"},
		"k2": {"v2", "v3"},
	}

	expectedMockStreamRoute := map[MessageStreamID][]MessageStreamID{
		StreamRegistry.GetStreamID("k1"): {StreamRegistry.GetStreamID("v1")},
		StreamRegistry.GetStreamID("k2"): {StreamRegistry.GetStreamID("v2"), StreamRegistry.GetStreamID("v3")},
	}

	value, err := mockPluginCfgReader.GetStreamRoutes("routes", make(map[MessageStreamID][]MessageStreamID))
	expect.NoError(err)
	expect.Equal(value, map[MessageStreamID][]MessageStreamID{})

	mockPluginCfg.Override("routes", mockStreamRoute)
	value, err = mockPluginCfgReader.GetStreamRoutes("routes", make(map[MessageStreamID][]MessageStreamID))
	expect.NoError(err)
	expect.Equal(value, expectedMockStreamRoute)
}

// Function gets an int value for a key or default value if non-existent
// Plan:
//  Create a new PluginConfig
//  add a key and an int value in the Settings
//  get the value back and Assert
func TestPluginConfigGetInt(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockPluginCfg := NewPluginConfig("", "core.mockPlugin")
	mockPluginCfgReader := NewPluginConfigReaderWithError(&mockPluginCfg)

	value, err := mockPluginCfgReader.GetInt("intkey", 0)
	expect.NoError(err)
	expect.Equal(int64(0), value)

	mockPluginCfg.Override("intkey", 2)
	value, err = mockPluginCfgReader.GetInt("intkey", 0)
	expect.NoError(err)
	expect.Equal(int64(2), value)
}

// Function gets an bool value for a key or default if non-existent
// Plan: similar to TestPluginConfigGetInt
func TestPluginConfigGetBool(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockPluginCfg := NewPluginConfig("", "core.mockPlugin")
	mockPluginCfgReader := NewPluginConfigReaderWithError(&mockPluginCfg)

	value, err := mockPluginCfgReader.GetBool("boolkey", false)
	expect.NoError(err)
	expect.Equal(value, false)

	mockPluginCfg.Override("boolkey", true)
	value, err = mockPluginCfgReader.GetBool("boolkey", false)
	expect.NoError(err)
	expect.Equal(value, true)
}

// Function gets a value for a key which is neither int or bool. Value encapsulated by interface
// Plan: similar to TestPluginConfigGetInt
func TestPluginConfigGetValue(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockPluginCfg := NewPluginConfig("", "core.mockPlugin")
	mockPluginCfgReader := NewPluginConfigReaderWithError(&mockPluginCfg)

	// get string value
	expect.Equal(mockPluginCfgReader.GetValue("valstrkey", ""), "")
	mockPluginCfg.Override("valstrkey", "valStr")
	expect.Equal(mockPluginCfgReader.GetValue("valstrkey", ""), "valStr")

	// get int value
	expect.Equal(mockPluginCfgReader.GetValue("valintkey", 0), 0)
	mockPluginCfg.Override("valintkey", 1)
	expect.Equal(mockPluginCfgReader.GetValue("valintkey", 0), 1)

	// get bool value
	expect.Equal(mockPluginCfgReader.GetValue("valboolkey", false), false)
	mockPluginCfg.Override("valboolkey", true)
	expect.Equal(mockPluginCfgReader.GetValue("valboolkey", false), true)

	// get a custom struct
	type CustomStruct struct {
		IntKey  int
		StrKey  string
		BoolKey bool
	}

	mockStruct := CustomStruct{1, "hello", true}
	//not sure if equal will do the trick here so, manual
	defaultCStruct := CustomStruct{0, "", false}
	defaultCStructRet := mockPluginCfgReader.GetValue("cstruct", defaultCStruct)
	ret, ok := defaultCStructRet.(CustomStruct)
	expect.True(ok)
	if ok {
		expect.Equal(defaultCStruct.IntKey, ret.IntKey)
		expect.Equal(defaultCStruct.BoolKey, ret.BoolKey)
		expect.Equal(defaultCStruct.StrKey, ret.StrKey)
	}

	mockPluginCfg.Override("cstruct", mockStruct)
	mockStructRet := mockPluginCfgReader.GetValue("cstruct", defaultCStruct)
	ret, ok = mockStructRet.(CustomStruct)
	expect.True(ok)
	if ok {
		expect.Equal(mockStruct.IntKey, ret.IntKey)
		expect.Equal(mockStruct.BoolKey, ret.BoolKey)
		expect.Equal(mockStruct.StrKey, ret.StrKey)
	}
}

func TestPluginConfigGetPlugin(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockPluginCfg := NewPluginConfig("mockPluginConfig", "core.mockPlugin")
	mockPluginCfgReader := NewPluginConfigReaderWithError(&mockPluginCfg)

	_, err := mockPluginCfgReader.GetPlugin("pluginkey", "", tcontainer.NewMarshalMap())
	expect.NotNil(err)

	mockPluginCfg.Override("pluginkey1", "core.mockPlugin")
	_, err = mockPluginCfgReader.GetPlugin("pluginkey1", "", tcontainer.NewMarshalMap())
	expect.NoError(err)

	mockPluginCfg.Override("pluginkey2", tcontainer.MarshalMap{"Type": "core.mockPlugin"})
	_, err = mockPluginCfgReader.GetPlugin("pluginkey2", "", tcontainer.NewMarshalMap())
	expect.NoError(err)
}

func TestPluginConfigSuggest(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockPluginCfg := NewPluginConfig("", "core.mockPlugin")
	mockPluginCfg.Override("Read", "foo")
	mockPluginCfg.Override("Write", "foo")

	mockPluginCfgReader := NewPluginConfigReader(&mockPluginCfg)

	readVal := mockPluginCfgReader.GetString("Read", "")
	expect.Equal("foo", readVal)
	writeVal := mockPluginCfgReader.GetString("Write", "")
	expect.Equal("foo", writeVal)

	suggest1 := mockPluginCfg.suggestKey("Reader")
	expect.Equal("Read", suggest1)
	suggest2 := mockPluginCfg.suggestKey("read")
	expect.Equal("Read", suggest2)
	suggest3 := mockPluginCfg.suggestKey("wirte")
	expect.Equal("Write", suggest3)
}
