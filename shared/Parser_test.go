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

package shared

import (
	"fmt"
	"testing"
)

const (
	testStateSearchName  = ">name"
	testStateName        = "name"
	testStateSearchValue = ">value"
	testStateValue       = "value"
	testStateArray       = "array"
)

type parserTestState struct {
	currentName string
	parsed      map[string]interface{}
}

func (t *parserTestState) parsedName(data []byte, state ParserStateID) {
	fmt.Println("name: ", string(data))
	t.currentName = string(data)
}

func (t *parserTestState) parsedValue(data []byte, state ParserStateID) {
	fmt.Println("value: ", string(data))
	t.parsed[t.currentName] = string(data)
}

func (t *parserTestState) parsedArray(data []byte, state ParserStateID) {
	fmt.Println("array: ", string(data))
	if value, exists := t.parsed[t.currentName]; !exists {
		t.parsed[t.currentName] = []string{string(data)}
	} else {
		array := value.([]string)
		t.parsed[t.currentName] = append(array, string(data))
	}
}

func TestTrie(t *testing.T) {
	expect := NewExpect(t)

	root := NewTrie([]byte("abcd"), new(int))
	root = root.Add([]byte("abd"), new(int))
	root = root.Add([]byte("cde"), new(int))

	node := root.Match([]byte("abcd"))
	if expect.True(node != nil) {
		expect.True(node.Payload != nil)
		expect.IntEq(4, node.PathLen)
	}

	node = root.Match([]byte("ab"))
	expect.True(node == nil)

	node = root.MatchStart([]byte("abcdef"))
	if expect.True(node != nil) {
		expect.True(node.Payload != nil)
		expect.IntEq(4, node.PathLen)
	}

	node = root.MatchStart([]byte("bcde"))
	expect.True(node == nil)

	root = NewTrie([]byte("a"), new(int))
	root = root.Add([]byte("b"), new(int))
	root = root.Add([]byte("c"), new(int))

	node = root.Match([]byte("c"))
	if expect.True(node != nil) {
		expect.True(node.Payload != nil)
		expect.IntEq(1, node.PathLen)
	}
}

func TestParser(t *testing.T) {
	state := parserTestState{parsed: make(map[string]interface{})}
	expect := NewExpect(t)

	dir := []TransitionDirective{
		TransitionDirective{testStateSearchName, `"`, testStateName, 0, nil},
		TransitionDirective{testStateSearchName, `}`, "", 0, nil},
		TransitionDirective{testStateName, `"`, testStateSearchValue, 0, state.parsedName},
		TransitionDirective{testStateSearchValue, `:`, testStateValue, 0, nil},
		TransitionDirective{testStateValue, `[`, testStateArray, 0, nil},
		TransitionDirective{testStateValue, `,`, testStateSearchName, 0, state.parsedValue},
		TransitionDirective{testStateValue, `}`, "", 0, state.parsedValue},
		TransitionDirective{testStateArray, `,`, testStateArray, 0, state.parsedArray},
		TransitionDirective{testStateArray, `],`, testStateSearchName, 0, state.parsedArray},
	}

	parser := NewTransitionParser()
	parser.AddDirectives(dir)

	dataTest := `{"test":123,"array":[a,b,c],"end":456}`
	parser.Parse([]byte(dataTest), testStateSearchName)

	expect.MapSet(state.parsed, "test")
	expect.MapSet(state.parsed, "array")
	expect.MapSet(state.parsed, "end")

	expect.MapSetStrEq(state.parsed, "test", "123")
	expect.MapSetStrArrayEq(state.parsed, "array", []string{"a", "b", "c"})
	expect.MapSetStrEq(state.parsed, "end", "456")
}
