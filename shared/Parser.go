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

// ParserFlag is an enum type for flags used in parser transitions.
type ParserFlag int

// ParserStateID is used as an integer-based reference to a specific parser state.
// You can use any number (e.g. a hash) as a parser state representative.
type ParserStateID uint32

// ParsedFunc defines a function that a token has been matched
type ParsedFunc func([]byte, ParserStateID)

const (
	// ParserFlagContinue continues parsing at the position of the match.
	// By default the matched token will be skipped. This flag prevents the
	// default behavior. In addition to that the parser will add the parsed
	// token to the value of the next match.
	ParserFlagContinue = ParserFlag(1 << iota)

	// ParserFlagAppend causes the parser to keep the current value for the next
	// match. By default a value will be restarted after each match. This flag
	// prevents the default behavior.
	ParserFlagAppend = ParserFlag(1 << iota)

	// ParserFlagInclude includes the matched token in the read value.
	// By default the value does not contain the matched token.
	ParserFlagInclude = ParserFlag(1 << iota)

	// ParserStateStop defines a state that causes the parsing to stop when
	// transitioned into.
	ParserStateStop = ParserStateID(0xFFFFFFFF)
)

// Transition defines a token based state change
type Transition struct {
	nextState ParserStateID
	flags     ParserFlag
	callback  ParsedFunc
}

// TransitionDirective contains a transition description that can be passed to
// the AddDirectives functions.
type TransitionDirective struct {
	State     string
	Token     string
	NextState string
	Flags     ParserFlag
	Callback  ParsedFunc
}

// TransitionParser defines the behavior of a parser by storing transitions from
// one state to another.
type TransitionParser struct {
	lookup []string
	tokens []*TrieNode
}

// NewTransition creates a new transition to a given state.
func NewTransition(nextState ParserStateID, flags ParserFlag, callback ParsedFunc) Transition {
	return Transition{
		nextState: nextState,
		flags:     flags,
		callback:  callback,
	}
}

// NewTransitionParser creates a new transition based parser
func NewTransitionParser() TransitionParser {
	return TransitionParser{
		lookup: []string{},
		tokens: []*TrieNode{},
	}
}

// GetStateID creates a hash from the given state name.
// Empty state names will be translated to ParserStateStop.
func (parser *TransitionParser) GetStateID(stateName string) ParserStateID {
	if len(stateName) == 0 {
		return ParserStateStop
	}

	for id, name := range parser.lookup {
		if name == stateName {
			return ParserStateID(id) // ### return, found ###
		}
	}

	id := ParserStateID(len(parser.lookup))
	parser.lookup = append(parser.lookup, stateName)
	parser.tokens = append(parser.tokens, nil)
	return id
}

// GetStateName returns the name for the given state id or an empty string if
// the id could not be found.
func (parser *TransitionParser) GetStateName(id ParserStateID) string {
	if id < ParserStateID(len(parser.lookup)) {
		return parser.lookup[id]
	}
	return ""
}

// AddDirectives is a convenience function to add multiple transitions in as a
// batch.
func (parser *TransitionParser) AddDirectives(directives []TransitionDirective) {
	for _, dir := range directives {
		parser.Add(dir.State, dir.Token, dir.NextState, dir.Flags, dir.Callback)
	}
}

// Add adds a new transition to a given parser state.
func (parser *TransitionParser) Add(stateName string, token string, nextStateName string, flags ParserFlag, callback ParsedFunc) {
	nextStateID := parser.GetStateID(nextStateName)
	parser.AddTransition(stateName, NewTransition(nextStateID, flags, callback), token)
}

// Stop adds a stop transition to a given parser state.
func (parser *TransitionParser) Stop(stateName string, token string, flags ParserFlag, callback ParsedFunc) {
	parser.AddTransition(stateName, NewTransition(ParserStateStop, flags, callback), token)
}

// AddTransition adds a transition from a given state to the map
func (parser *TransitionParser) AddTransition(stateName string, newTrans Transition, token string) {
	stateID := parser.GetStateID(stateName)

	if state := parser.tokens[stateID]; state == nil {
		parser.tokens[stateID] = NewTrie([]byte(token), newTrans)
	} else {
		parser.tokens[stateID] = state.Add([]byte(token), newTrans)
	}
}

// Parse starts parsing at a given stateID.
// This function returns the remaining parts of data that did not match a
// transition as well as the last state the parser has been set to.
func (parser TransitionParser) Parse(data []byte, state string) ([]byte, ParserStateID) {
	currentStateID := parser.GetStateID(state)
	currentState := parser.tokens[currentStateID]
	dataLen := len(data)
	readStartIdx := 0
	continueIdx := 0

	for parseIdx := 0; parseIdx < dataLen; parseIdx++ {
		node := currentState.MatchStart(data[parseIdx:])
		if node == nil {
			continue // ### continue, no match ###
		}

		t := node.Payload.(Transition)
		if t.callback != nil {
			if t.flags&ParserFlagInclude != 0 {
				t.callback(data[readStartIdx:parseIdx+node.PathLen], currentStateID)
			} else {
				t.callback(data[readStartIdx:parseIdx], currentStateID)
			}
		}

		if t.flags&ParserFlagContinue == 0 {
			parseIdx += node.PathLen - 1
			continueIdx = parseIdx + 1
		} else {
			continueIdx = parseIdx
		}

		if t.flags&ParserFlagAppend == 0 {
			readStartIdx = continueIdx
		}

		currentStateID = t.nextState
		if currentStateID == ParserStateStop {
			break // ### break, stop state ###
		}

		currentState = parser.tokens[currentStateID]
	}

	if readStartIdx == dataLen {
		return nil, currentStateID // ### return, everything parsed ###
	}

	return data[readStartIdx:], currentStateID
}
