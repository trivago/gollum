package shared

import (
	"fmt"
	"strings"
	"testing"
)

const (
	testParserStateID = iota
	testParserStateType
	testParserStateTypeArray
	testParserStateArraySize
	testParserStateData
	testParserStateDataArray
)

var testParserStateNames = []string{
	"ID",
	"Type",
	"Data",
}

var testParserTransitions = [][]Transition{
	/* ID        */ {NewTransition(":", testParserStateType, ParserFlagDone)},
	/* Type      */ {NewTransition("[", testParserStateArraySize, ParserFlagRestartAfter), NewTransition(":", testParserStateData, ParserFlagDone)},
	/* TypeArray */ {NewTransition(":", testParserStateDataArray, ParserFlagDone)},
	/* ArraySize */ {NewTransition("]", testParserStateTypeArray, ParserFlagDone)},
	/* Data      */ {NewTransition(";", testParserStateID, ParserFlagDone)},
	/* DataArray */ {NewTransition(",", testParserStateDataArray, ParserFlagDone), NewTransition(";", testParserStateID, ParserFlagDone)},
}

func TestParser(t *testing.T) {
	expect := NewExpect(t)
	parser := NewParser(testParserTransitions)

	ids := []string{"Value1", "Value2"}
	types := []string{"[3]int", "string"}
	data := []string{"1,2,3", "hello world"}

	testString := ""
	for i := 0; i < len(ids); i++ {
		testString += fmt.Sprintf("%s:%s:%s;", ids[i], types[i], data[i])
	}

	result := parser.Parse([]byte(testString), testParserStateID)
	expect.IntEq(9, len(result))

	idx := 0
	arraySize := uint64(0)
	arrayData := ""

	for _, v := range result {
		switch v.State {
		case testParserStateID:
			expect.StringEq(ids[idx], string(v.Data))

		case testParserStateType:
			expect.StringEq(types[idx], string(v.Data))

		case testParserStateTypeArray:
			typeNameStart := strings.Index(types[idx], "]") + 1
			expect.StringEq(types[idx][typeNameStart:], string(v.Data))
			arrayData = ""

		case testParserStateArraySize:
			expect.StringEq("3", string(v.Data))
			arraySize, _ = Btoi(v.Data)

		case testParserStateData:
			expect.StringEq(data[idx], string(v.Data))
			idx++

		case testParserStateDataArray:
			arrayData += string(v.Data)
			arraySize--
			if arraySize == 0 {
				expect.StringEq(data[idx], arrayData)
				idx++
			} else {
				arrayData += ","
			}
		}
	}

	expect.IntEq(2, idx)
}
