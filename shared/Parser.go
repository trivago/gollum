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
	"math"
)

// ParserFlag is an enum type for flags used in parser transitions.
type ParserFlag int

const (
	// ParserFlagRestart does not process the read bytes and starts reading from
	// the start position of the match.
	ParserFlagRestart = ParserFlag(iota)

	// ParserFlagPersist creates an parsed entry for the current state from
	// start of reading to the current parsing position (unless ParserFlagInclude)
	// is set.
	ParserFlagPersist = ParserFlag(iota)

	// ParserFlagSkip continues parsing after the matched token.
	ParserFlagSkip = ParserFlag(1 << iota)

	// ParserFlagContinue prevents the read-start position to be reset after a
	// token has been matched.
	ParserFlagContinue = ParserFlag(1 << iota)

	// ParserFlagDone is the standard flag used when a token has been matched.
	// Same as ParserFlagPersist | ParserFlagSkip.
	ParserFlagDone = ParserFlagPersist | ParserFlagSkip

	// ParserFlagIgnore should be used for tokens the should be skipped during
	// parsing but contain information that belong to the current state.
	// Same as ParserFlagContinue | ParserFlagSkip.
	ParserFlagIgnore = ParserFlagContinue | ParserFlagSkip

	// ParserFlagRestartAfter does not process the read bytes and start reading
	// form the position after the match
	ParserFlagRestartAfter = ParserFlagRestart | ParserFlagSkip

	// ParserStateStop defines a state that causes the parsing to stop when
	// transitioned into.
	ParserStateStop = math.MaxInt32
)

// Transition defines a token based state change
type Transition struct {
	token    []byte
	tokenLen int
	state    int
	flags    ParserFlag
}

// StateData contains the slice parsed for a given state
type StateData struct {
	Data  []byte
	State int
}

// Parser is the main struct for token based parsing
type Parser struct {
	state           int
	transitions     [][]Transition
	candidateBuffer []byte
}

// NewTransition creates a new transition object to be used with NewParser.
func NewTransition(token string, state int, flags ParserFlag) Transition {
	return Transition{
		token:    []byte(token),
		tokenLen: len(token),
		state:    state,
		flags:    flags,
	}
}

// NewParser generates a new parser based on a set of given transitions
func NewParser(transitions [][]Transition) Parser {
	maxNumTokens := 0
	for _, state := range transitions {
		numTokens := len(state)
		if numTokens > maxNumTokens {
			maxNumTokens = numTokens
		}
	}

	return Parser{
		state:           0,
		transitions:     transitions,
		candidateBuffer: make([]byte, maxNumTokens),
	}
}

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
