package shared

import (
	"bytes"
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
	state       int
	transitions [][]Transition
}

// NewParser generates a new parser based on a set of given transitions
func NewParser(transitions [][]Transition) Parser {
	return Parser{
		state:       0,
		transitions: transitions,
	}
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

// Parse parses a string with the transition passed to the parser object.
func (parser Parser) Parse(message []byte, initialState int) []StateData {
	var result []StateData

	startIdx := 0
	parser.state = initialState
	messageLen := len(message)

	for parseIdx := 0; parseIdx < messageLen && parser.state != ParserStateStop; parseIdx++ {

		// Note: If all compares fail we could move by the length of the smallest
		//       non-repeating ident part, but this usually is one char anyway.
		for _, t := range parser.transitions[parser.state] {
			cmpIdxEnd := parseIdx + t.tokenLen
			if cmpIdxEnd > messageLen {
				continue
			}

			if bytes.Equal(message[parseIdx:cmpIdxEnd], t.token) {
				//fmt.Printf("[%s] %d %d [%s]", string(message[parseIdx:cmpIdxEnd]), startIdx, parseIdx, string(message[startIdx:parseIdx]))

				if t.flags&ParserFlagPersist != 0 {
					//fmt.Print(" w")
					result = append(result, StateData{
						Data:  message[startIdx:parseIdx],
						State: parser.state,
					})
				}

				// Move the iterator over the matched element (increment after loop)
				if t.flags&ParserFlagSkip != 0 {
					//fmt.Print(" s")
					parseIdx += t.tokenLen - 1
				}

				// Restart the slice if continue is NOT set
				if t.flags&ParserFlagContinue == 0 {
					//fmt.Print(" n")
					startIdx = parseIdx + 1
				}

				parser.state = t.state
				//fmt.Print("\n")
				break
			}
		}
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
