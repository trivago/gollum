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
	mockPluginCfg := NewPluginConfig("", "mockPlugin")
	// Note that thes values have to be lowercase
	mockPluginCfg.Override("stringkey", "value")
	mockPluginCfg.Override("number", 1)

	// access one field
	sValue := mockPluginCfg.GetString("stringKey", "")
	expect.Equal(sValue, "value")
	expect.False(mockPluginCfg.Validate())

	// access second one
	iValue := mockPluginCfg.GetInt("number", 0)
	expect.Equal(iValue, 1)
	expect.True(mockPluginCfg.Validate())
}

// Function reads initializes pluginConfig with predefined values and
// non-predefined values in the Settings
// Plan:
//  Create a new PluginConfig
//  Create a new tgo.MarshalMap with mock key values
//  Check if they are actually added, if they are, assert the key-value correctness
func TestPluginConfigRead(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockPluginCfg := NewPluginConfig("", "mockPlugin")

	// create a mock MarshalMap
	testMarshalMap := tcontainer.NewMarshalMap()
	testMarshalMap["instances"] = 0

	mockPluginCfg.Read(testMarshalMap)
	// with 0 instance, plugin should be disabled
	expect.False(mockPluginCfg.Enable)

	//reset mockPluginCfg
	mockPluginCfg = NewPluginConfig("", "mockPlugin")
	testMarshalMap["Enable"] = true
	testMarshalMap["Instances"] = 2
	testMarshalMap["Host"] = "someHost"
	testMarshalMap["Database"] = "someDatabase"

	mockPluginCfg.Read(testMarshalMap)

	// Check for the bundled config options
	expect.True(mockPluginCfg.Enable)
	expect.Equal(mockPluginCfg.Instances, 2)

	// check for the miscelleneous settings key added
	expect.Equal(mockPluginCfg.GetString("host", ""), "someHost")
	expect.Equal(mockPluginCfg.GetString("database", ""), "someDatabase")
}

// Function checks if predefined value exists or not
// Plan:
//  Create a new PluginConfig
//  Add non-predefined Values
//  Check if added ones return true, and others false
func TestPluginConfigHasValue(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockPluginCfg := NewPluginConfig("", "mockPlugin")

	expect.False(mockPluginCfg.HasValue("akey"))

	mockPluginCfg.Override("aKey", 1)
	expect.True(mockPluginCfg.HasValue("akey"))
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
	mockPluginCfg := NewPluginConfig("", "mockPlugin")

	mockPluginCfg.Override("akey", "aValue")
	// make sure the value exists
	expect.True(mockPluginCfg.HasValue("akey"))

	// TODO: maybe check for the validate here. Since the value is already checked

	expect.Equal(mockPluginCfg.GetString("akey", ""), "aValue")

	// override
	mockPluginCfg.Override("akey", "newValue")

	//check for new value
	expect.Equal(mockPluginCfg.GetString("akey", ""), "newValue")

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
	mockPluginCfg := NewPluginConfig("", "mockPlugin")

	//check for non-existant key
	expect.Equal(mockPluginCfg.GetString("akey", "default"), "default")

	mockPluginCfg.Override("aKey", "aValue")
	expect.Equal(mockPluginCfg.GetString("akey", "default"), "aValue")
}

// Function gets the string array for a key or default value if non existant
// Plan: Similart to TestPluginConfigGetString
func TestPluginConfigGetStringArray(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockPluginCfg := NewPluginConfig("", "mockPlugin")

	mockStringArray := []string{"el1", "el2", "el3"}

	// default value is returned
	expect.Equal(len(mockPluginCfg.GetStringArray("arrkey", []string{})), 0)

	mockPluginCfg.Override("arrKey", mockStringArray)
	//since expect.Equal is doing reflect.deepValueEqual, arrays should be properly compared
	expect.Equal(mockPluginCfg.GetStringArray("arrkey", []string{}), mockStringArray)

}

// Function gets the stringMap for a key or default value if not existant
// Plan: Similar to TestPluginConfigGetString but the Map structure needs assertion
func TestPluginConfigGetStringMap(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockPluginCfg := NewPluginConfig("", "mockPlugin")

	mockStringMap := map[string]string{
		"k1": "v1",
		"k2": "v2",
		"k3": "v3",
	}

	expect.Equal(len(mockPluginCfg.GetStringMap("strmapkey", map[string]string{})), 0)

	mockPluginCfg.Override("strmapkey", mockStringMap)
	expect.Equal(mockPluginCfg.GetStringMap("strmapkey", map[string]string{}), mockStringMap)
}

// Function gets an array of MessageStreamID for a given key
// Plan:
//  Create a new PluginConfig
//  Add an array of streams with mocked values in Settings
//  get the streamArray
//  Check the hash received hash with manual generation from streamregistery.getStreamID
func TestPluginConfigGetStreamArray(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockPluginCfg := NewPluginConfig("", "mockPlugin")

	mockStreamArray := []string{
		"stream1",
		"stream2",
	}

	mockStreamHashed := []MessageStreamID{
		GetStreamID("stream1"),
		GetStreamID("stream2"),
	}

	expect.Equal(len(mockPluginCfg.GetStreamArray("mockstream", []MessageStreamID{})), 0)

	mockPluginCfg.Override("mockstream", mockStreamArray)
	expect.Equal(mockPluginCfg.GetStreamArray("mockstream", []MessageStreamID{}), mockStreamHashed)

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
	mockPluginCfg := NewPluginConfig("", "mockPlugin")
	defaultValue := "v0"

	mockStringMap := map[string]string{
		"k1": "v1",
		"k2": "v2",
		"k3": "v3",
	}

	expectedMapWithWildcard := map[MessageStreamID]string{
		WildcardStreamID:  defaultValue,
		GetStreamID("k1"): "v1",
		GetStreamID("k2"): "v2",
		GetStreamID("k3"): "v3",
	}

	expectedMapWithoutWildcard := map[MessageStreamID]string{
		GetStreamID("k1"): "v1",
		GetStreamID("k2"): "v2",
		GetStreamID("k3"): "v3",
	}

	expectedMapOnlyWildCard := map[MessageStreamID]string{
		WildcardStreamID: defaultValue,
	}

	// should be empty because default value is empty, wildcard returned and the key doesn't exist
	expect.Equal(mockPluginCfg.GetStreamMap("streammap", ""), map[MessageStreamID]string{})

	// should return wildcard stream when default value not empty and the key doesn't exist
	expect.Equal(mockPluginCfg.GetStreamMap("streammap", defaultValue), expectedMapOnlyWildCard)

	mockPluginCfg.Override("streammap", mockStringMap)
	// without default value, hashed map without wildcard should be returned
	expect.Equal(mockPluginCfg.GetStreamMap("streammap", ""), expectedMapWithoutWildcard)
	// with default value, hashed map with wildcard should be returned
	expect.Equal(mockPluginCfg.GetStreamMap("streammap", defaultValue), expectedMapWithWildcard)

}

// Function gets streamRoutes where key is a string and value is a streamID
// Plan: similar to TestPluginConfigGetStreamMap with streamId and value swapped
func TestPluginConfigGetStreamRoutes(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockPluginCfg := NewPluginConfig("", "mockPlugin")

	mockStreamRoute := map[string][]string{
		"k1": []string{"v1"},
		"k2": []string{"v2", "v3"},
	}

	expectedMockStreamRoute := map[MessageStreamID][]MessageStreamID{
		GetStreamID("k1"): []MessageStreamID{GetStreamID("v1")},
		GetStreamID("k2"): []MessageStreamID{GetStreamID("v2"), GetStreamID("v3")},
	}

	expect.Equal(mockPluginCfg.GetStreamRoutes("routes"), map[MessageStreamID][]MessageStreamID{})
	mockPluginCfg.Override("routes", mockStreamRoute)
	expect.Equal(mockPluginCfg.GetStreamRoutes("routes"), expectedMockStreamRoute)
}

// Function gets an int value for a key or default value if non-existant
// Plan:
//  Create a new PluginConfig
//  add a key and an int value in the Settings
//  get the value back and Assert
func TestPluginConfigGetInt(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockPluginCfg := NewPluginConfig("", "mockPlugin")

	expect.Equal(mockPluginCfg.GetInt("intkey", 0), 0)
	mockPluginCfg.Override("intkey", 2)
	expect.Equal(mockPluginCfg.GetInt("intkey", 0), 2)
}

// Function gets an bool value for a key or default if non-existant
// Plan: similar to TestPluginConfigGetInt
func TestPluginConfigGetBool(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockPluginCfg := NewPluginConfig("", "mockPlugin")

	expect.Equal(mockPluginCfg.GetBool("boolkey", false), false)
	mockPluginCfg.Override("boolkey", true)
	expect.Equal(mockPluginCfg.GetBool("boolkey", false), true)
}

// Function gets a value for a key which is neither int or bool. Value encapsulated by interface
// Plan: similart to TestPluginConfigGetInt
func TestPluginConfigGetValue(t *testing.T) {
	expect := ttesting.NewExpect(t)
	mockPluginCfg := NewPluginConfig("", "mockPlugin")

	// get string value
	expect.Equal(mockPluginCfg.GetValue("valstrkey", ""), "")
	mockPluginCfg.Override("valstrkey", "valStr")
	expect.Equal(mockPluginCfg.GetValue("valstrkey", ""), "valStr")

	// get int value
	expect.Equal(mockPluginCfg.GetValue("valintkey", 0), 0)
	mockPluginCfg.Override("valintkey", 1)
	expect.Equal(mockPluginCfg.GetValue("valintkey", 0), 1)

	// get bool value
	expect.Equal(mockPluginCfg.GetValue("valboolkey", false), false)
	mockPluginCfg.Override("valboolkey", true)
	expect.Equal(mockPluginCfg.GetValue("valboolkey", false), true)

	// get a custom struct
	type CustomStruct struct {
		IntKey  int
		StrKey  string
		BoolKey bool
	}

	mockStruct := CustomStruct{1, "hello", true}
	//not sure if equal will do the trick here so, manual
	defaultCStruct := CustomStruct{0, "", false}
	defaultCStructRet := mockPluginCfg.GetValue("cstruct", defaultCStruct)
	ret, ok := defaultCStructRet.(CustomStruct)
	expect.True(ok)
	if ok {
		expect.Equal(defaultCStruct.IntKey, ret.IntKey)
		expect.Equal(defaultCStruct.BoolKey, ret.BoolKey)
		expect.Equal(defaultCStruct.StrKey, ret.StrKey)
	}

	mockPluginCfg.Override("cstruct", mockStruct)
	mockStructRet := mockPluginCfg.GetValue("cstruct", defaultCStruct)
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
	mockPluginCfg := NewPluginConfig("", "mockPlugin")

	_, err := mockPluginCfg.GetPlugin("pluginkey", "", tcontainer.NewMarshalMap())
	expect.NotNil(err)

	mockPluginCfg.Override("pluginkey1", "core.mockPlugin")
	_, err = mockPluginCfg.GetPlugin("pluginkey1", "", tcontainer.NewMarshalMap())
	expect.NoError(err)

	mockPluginCfg.Override("pluginkey2", tcontainer.MarshalMap{"type": "core.mockPlugin"})
	_, err = mockPluginCfg.GetPlugin("pluginkey2", "", tcontainer.NewMarshalMap())
	expect.NoError(err)
}
