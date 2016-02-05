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

package shared

import (
	"testing"
)

func TestMarshalMapBaseTypes(t *testing.T) {
	expect := NewExpect(t)
	testMap := NewMarshalMap()

	testMap["t1"] = 10
	t1, err := testMap.Int("t1")
	expect.NoError(err)
	expect.Equal(t1, 10)

	testMap["t2"] = uint64(10)
	t2, err := testMap.Uint64("t2")
	expect.NoError(err)
	expect.Equal(t2, uint64(10))

	testMap["t3"] = float64(10)
	t3, err := testMap.Float64("t3")
	expect.NoError(err)
	expect.Equal(t3, float64(10))

	testMap["t4"] = "test"
	t4, err := testMap.String("t4")
	expect.NoError(err)
	expect.Equal(t4, "test")
}

func TestMarshalMapArrays(t *testing.T) {
	expect := NewExpect(t)
	testMap := NewMarshalMap()

	iArray := []interface{}{"test", 0}
	testMap["a1"] = iArray
	a1, err := testMap.Array("a1")
	expect.NoError(err)
	expect.Equal(a1, iArray)

	sArray := []string{"test", "test", "test"}
	testMap["a2"] = sArray
	a2, err := testMap.StringArray("a2")
	expect.NoError(err)
	expect.Equal(a2, sArray)
}

func TestMarshalMapMaps(t *testing.T) {
	expect := NewExpect(t)
	testMap := NewMarshalMap()

	iiMap := make(map[interface{}]interface{})
	iiMap["0"] = "test"
	iiMap[1] = 0
	testMap["m1"] = iiMap
	m1, err := testMap.Map("m1")
	expect.NoError(err)
	expect.Equal(m1, iiMap)
}

func TestMarshalMapMarshalMaps(t *testing.T) {
	expect := NewExpect(t)
	testMap := NewMarshalMap()

	siMap := NewMarshalMap()
	siMap["test1"] = "test"
	siMap["test2"] = 0
	testMap["m2"] = siMap
	m2, err := testMap.MarshalMap("m2")
	expect.NoError(err)
	expect.Equal(m2, siMap)

	siMap2 := make(map[string]interface{})
	siMap2["test1"] = siMap["test1"]
	siMap2["test2"] = siMap["test2"]
	testMap["m2a"] = siMap2
	m2a, err := testMap.MarshalMap("m2a")
	expect.NoError(err)
	expect.Equal(m2a, siMap)

	siMap3 := make(map[interface{}]interface{})
	siMap3["test1"] = siMap["test1"]
	siMap3["test2"] = siMap["test2"]
	testMap["m2b"] = siMap3
	m2b, err := testMap.MarshalMap("m2b")
	expect.NoError(err)
	expect.Equal(m2b, siMap)
}

func TestMarshalMapStringMaps(t *testing.T) {
	expect := NewExpect(t)
	testMap := NewMarshalMap()

	ssMap := make(map[string]string)
	ssMap["test1"] = "test"
	ssMap["test2"] = "test"
	testMap["m3"] = ssMap
	m3, err := testMap.StringMap("m3")
	expect.NoError(err)
	expect.Equal(m3, ssMap)

	ssMap2 := make(map[string]interface{})
	ssMap2["test1"] = ssMap["test1"]
	ssMap2["test2"] = ssMap["test1"]
	testMap["m3a"] = ssMap2
	m3a, err := testMap.StringMap("m3a")
	expect.NoError(err)
	expect.Equal(m3a, ssMap)

	ssMap3 := make(map[interface{}]interface{})
	ssMap3["test1"] = ssMap["test1"]
	ssMap3["test2"] = ssMap["test1"]
	testMap["m3b"] = ssMap3
	m3b, err := testMap.StringMap("m3b")
	expect.NoError(err)
	expect.Equal(m3b, ssMap)
}

func TestMarshalMapStringArrayMaps(t *testing.T) {
	expect := NewExpect(t)
	testMap := NewMarshalMap()

	// String array map

	sArray := []string{"test", "test", "test"}
	siArray := []interface{}{"test", "test", "test"}

	ssaMap := make(map[string][]string)
	ssaMap["test1"] = sArray
	ssaMap["test2"] = sArray
	testMap["m4"] = ssaMap
	m4, err := testMap.StringArrayMap("m4")
	expect.NoError(err)
	expect.Equal(m4, ssaMap)

	ssaMap2 := make(map[string]interface{})
	ssaMap2["test1"] = sArray
	ssaMap2["test2"] = sArray
	testMap["m4a"] = ssaMap2
	m4a, err := testMap.StringArrayMap("m4a")
	expect.NoError(err)
	expect.Equal(m4a, ssaMap)

	ssaMap3 := make(map[interface{}]interface{})
	ssaMap3["test1"] = sArray
	ssaMap3["test2"] = sArray
	testMap["m4b"] = ssaMap3
	m4b, err := testMap.StringArrayMap("m4b")
	expect.NoError(err)
	expect.Equal(m4b, ssaMap)

	ssaMap4 := make(map[interface{}][]interface{})
	ssaMap4["test1"] = siArray
	ssaMap4["test2"] = siArray
	testMap["m4c"] = ssaMap4
	m4c, err := testMap.StringArrayMap("m4c")
	expect.NoError(err)
	expect.Equal(m4c, ssaMap)
}

func TestMarshalMapPath(t *testing.T) {
	expect := NewExpect(t)
	testMap := NewMarshalMap()

	nestedMap1 := make(map[string]interface{})
	nestedMap1["f"] = "ok"

	nestedMap2 := make(map[string]interface{})
	nestedMap2["d"] = "ok"
	nestedMap2["e"] = nestedMap1

	nestedArray1 := []interface{}{
		"ok",
		nestedMap2,
	}

	nestedArray2 := []interface{}{
		"ok",
		nestedMap2,
		nestedArray1,
	}

	testMap["a"] = "ok"
	testMap["b"] = nestedArray2
	testMap["c"] = nestedMap2

	val, valid := testMap.Path("a")
	expect.True(valid)
	expect.Equal("ok", val)

	val, valid = testMap.Path("b")
	expect.True(valid)

	val, valid = testMap.Path("c")
	expect.True(valid)

	val, valid = testMap.Path("b[0]")
	expect.True(valid)
	expect.Equal("ok", val)

	val, valid = testMap.Path("b[1]d")
	expect.True(valid)
	expect.Equal("ok", val)

	val, valid = testMap.Path("b[1]e/f")
	expect.True(valid)
	expect.Equal("ok", val)

	val, valid = testMap.Path("b[2][0]")
	expect.True(valid)
	expect.Equal("ok", val)

	val, valid = testMap.Path("b[2][1]d")
	expect.True(valid)
	expect.Equal("ok", val)

	val, valid = testMap.Path("b[2][1]e/f")
	expect.True(valid)
	expect.Equal("ok", val)

	val, valid = testMap.Path("c/d")
	expect.True(valid)
	expect.Equal("ok", val)

	val, valid = testMap.Path("c/e/f")
	expect.True(valid)
	expect.Equal("ok", val)
}
