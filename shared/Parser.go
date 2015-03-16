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
	"hash/fnv"
)

// ParserFlag is an enum type for flags used in parser transitions.
type ParserFlag int

// ParserStateID is used as an integer-based reference to a specific parser state.
// You can use any number (e.g. a hash) as a parser state representative.
type ParserStateID uint32

// ParsedFunc defines a function that a token has been matched
type ParsedFunc func(data []byte)

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
	NextState string
	Token     string
	Flags     ParserFlag
	Callback  ParsedFunc
}

// TransitionParser defines the behavior of a parser by storing transitions from
// one state to another.
type TransitionParser map[ParserStateID]*TrieNode

// GetParserStateID creates a hash from the given state name.
// Empty state names will be translated to ParserStateStop.
func GetParserStateID(state string) ParserStateID {
	if state == "" {
		return ParserStateStop
	}
	hasher := fnv.New32a()
	hasher.Write([]byte(state))
	return ParserStateID(hasher.Sum32())
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
	return make(TransitionParser)
}

// AddDirectives is a convenience function to add multiple transitions in as a
// batch.
func (parser TransitionParser) AddDirectives(directives []TransitionDirective) {
	for _, dir := range directives {
		parser.Add(dir.State, dir.NextState, dir.Token, dir.Flags, dir.Callback)
	}
}

// Add adds a new transition to a given parser state.
func (parser TransitionParser) Add(stateName string, nextStateName string, token string, flags ParserFlag, callback ParsedFunc) {
	nextStateID := GetParserStateID(nextStateName)
	parser.AddTransition(stateName, NewTransition(nextStateID, flags, callback), token)
}

// Stop adds a stop transition to a given parser state.
func (parser TransitionParser) Stop(stateName string, token string, flags ParserFlag, callback ParsedFunc) {
	parser.AddTransition(stateName, NewTransition(ParserStateStop, flags, callback), token)
}

// AddTransition adds a transition from a given state to the map
func (parser TransitionParser) AddTransition(stateName string, newTrans Transition, token string) {
	stateID := GetParserStateID(stateName)

	if state, exists := parser[stateID]; exists {
		parser[stateID] = state.Add([]byte(token), newTrans)
	} else {
		parser[stateID] = NewTrie([]byte(token), newTrans)
	}
}

// ParseNamed is a alias for Parse(data, GetParserStateID(state)
func (parser TransitionParser) ParseNamed(data []byte, state string) ([]byte, ParserStateID) {
	initialStateID := GetParserStateID(state)
	return parser.Parse(data, initialStateID)
}

// Parse starts parsing at a given stateID.
// This function returns the remaining parts of data that did not match a
// transition as well as the last state the parser has been set to.
func (parser TransitionParser) Parse(data []byte, state ParserStateID) ([]byte, ParserStateID) {
	currentStateID := state
	currentState := parser[currentStateID]
	readStartIdx := 0

	for parseIdx := 0; parseIdx < len(data); parseIdx++ {
		node, length := currentState.MatchStart(data[parseIdx:])
		if node != nil {
			t := node.Payload.(Transition)

			if t.callback != nil {
				if t.flags&ParserFlagInclude != 0 {
					t.callback(data[readStartIdx : parseIdx+length])
				} else {
					t.callback(data[readStartIdx:parseIdx])
				}
			}

			continueIdx := parseIdx

			if t.flags&ParserFlagContinue == 0 {
				parseIdx += length - 1
				continueIdx = parseIdx + 1
			}

			if t.flags&ParserFlagAppend == 0 {
				readStartIdx = continueIdx
			}

			if t.nextState == ParserStateStop {
				break // ### break, stop state ###
			}

			currentStateID = t.nextState
			currentState = parser[currentStateID]
		}
	}

	if readStartIdx == len(data) {
		return nil, currentStateID // ### return, everything parsed ###
	}

	return data[readStartIdx:], currentStateID
}

/*
// Do a state transition, i.e. set the next state, return the new transition
// tokens and the number of tokens in the returned array.
func (parser *Parser) setState(state int) ([]Transition, int, int) {
	parser.state = state

	trans := parser.transitions[parser.state]
	numTrans := len(trans)
	numCandidates := 0

nextToken:
	for tIdx := 0; tIdx < numTrans; tIdx++ {
		firstChar := trans[tIdx].token[0]
		for cIdx := 0; cIdx < numCandidates; cIdx++ {
			if firstChar == parser.candidateBuffer[cIdx] {
				continue nextToken
			}
		}
		parser.candidateBuffer[numCandidates] = firstChar
		numCandidates++
	}

	return trans, numTrans, numCandidates
}

// Parse parses a string with the transition passed to the parser object.
func (parser Parser) Parse(message []byte, initialState int) []StateData {
	result := make([]StateData, 0, len(parser.transitions))

	if parser.state == ParserStateStop {
		return result
	}

	startIdx := 0
	messageLen := len(message)
	transitions, numTransitions, numCandidates := parser.setState(initialState)

parsing:
	// Iterate over the whole message
	for parseIdx := 0; parseIdx < messageLen; {

		// Fast test to check if we need to have a closer look at the tokens
		candidate := false
		for i := 0; i < numCandidates && !candidate; i++ {
			candidate = message[parseIdx] == parser.candidateBuffer[i]
		}

		if candidate {
		nextToken:
			// Check all possible transitions
			for tIdx := 0; tIdx < numTransitions; tIdx++ {
				t := &transitions[tIdx]
				cmpIdxEnd := parseIdx + t.tokenLen

				// Bounds check
				if cmpIdxEnd > messageLen {
					continue nextToken
				}

				// Check token match
				for i := 0; i < t.tokenLen; i++ {
					if message[parseIdx+i] != t.token[i] {
						continue nextToken
					}
				}

				//fmt.Printf("[%s] s%d p%d e%d +%d [%s]", string(message[parseIdx:cmpIdxEnd]), startIdx, parseIdx, cmpIdxEnd, stride, string(message[startIdx:parseIdx]))

				// Store the result
				if t.flags&ParserFlagPersist != 0 {
					//fmt.Print(" w")
					result = append(result, StateData{
						Data:  message[startIdx:parseIdx],
						State: parser.state,
					})
				}

				// Move the iterator over the matched element
				if t.flags&ParserFlagSkip != 0 {
					//fmt.Print(" s")
					parseIdx += t.tokenLen
				} else {
					parseIdx++
				}

				// Restart the slice if continue is NOT set
				if t.flags&ParserFlagContinue == 0 {
					//fmt.Print(" n")
					startIdx = parseIdx
				}

				//fmt.Print("\n")

				// If the next state is "stop" stop here at once
				if t.state == ParserStateStop {
					return result
				}

				transitions, numTransitions, numCandidates = parser.setState(t.state)
				continue parsing
			}
		}

		parseIdx++
	}

	// Store the remaining data
	if startIdx < messageLen {
		//fmt.Printf("[end] %d %d [%s] w\n", startIdx, messageLen, string(message[startIdx:]))
		result = append(result, StateData{
			Data:  message[startIdx:],
			State: parser.state,
		})
	}

	return result
}
*/
