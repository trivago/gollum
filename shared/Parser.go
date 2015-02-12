package shared

import (
	"bytes"
	"math"
)

type ParserFlag int

const (
	ParserFlagNop     = ParserFlag(iota)
	ParserFlagStart   = ParserFlag(iota)
	ParserFlagKeep    = ParserFlag(1 << iota)
	ParserFlagPersist = ParserFlag(1 << iota)
	ParserFlagDone    = ParserFlagPersist | ParserFlagStart
	ParserFlagNext    = ParserFlagPersist | ParserFlagKeep

	ParserStateStop = math.MaxInt32
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

parsing:
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
				} else if t.flags&ParserFlagKeep != 0 {
					startIdx = cmpStartIdx
					segmentLen = t.identLen
				}

				parser.state = t.state

				if t.state >= len(parser.transitions) {
					break parsing
				}
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
