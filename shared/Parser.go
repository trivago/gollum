package shared

import (
	"bytes"
)

type ParserFlag int

const (
	ParserFlagNop     = ParserFlag(0)
	ParserFlagStart   = ParserFlag(1)
	ParserFlagPersist = ParserFlag(2)
	ParserFlagDone    = ParserFlagPersist | ParserFlagStart
)

type Transition struct {
	ident    []byte
	identLen int
	state    int
	flags    ParserFlag
}

type StateData struct {
	Data  []byte
	State int
}

type Parser struct {
	state       int
	transitions [][]Transition
}

func NewParser(transitions [][]Transition) Parser {
	return Parser{
		state:       0,
		transitions: transitions,
	}
}

func NewTransition(ident string, state int, flags ParserFlag) Transition {
	return Transition{
		ident:    []byte(ident),
		identLen: len(ident),
		state:    state,
		flags:    flags,
	}
}

func (parser Parser) Parse(message []byte) []StateData {
	var result []StateData

	parser.state = 0
	startIdx := 0
	segmentLen := 0

	for parseIdx := 1; parseIdx <= len(message); parseIdx++ {
		segmentLen++

		for _, t := range parser.transitions[parser.state] {
			if segmentLen < t.identLen {
				continue
			}

			cmpStartIdx := parseIdx - t.identLen
			if bytes.Equal(message[cmpStartIdx:parseIdx], t.ident) {

				if t.flags&ParserFlagPersist != 0 {
					result = append(result, StateData{
						Data:  message[startIdx:cmpStartIdx],
						State: parser.state,
					})
				}

				if t.flags&ParserFlagStart != 0 {
					startIdx = parseIdx
					segmentLen = 0
				}

				if t.state >= len(parser.transitions) {
					return result
				}

				parser.state = t.state
				break
			}
		}
	}

	if len(message)-startIdx > 0 {
		result = append(result, StateData{
			Data:  message[startIdx:],
			State: parser.state,
		})
	}

	return result
}
