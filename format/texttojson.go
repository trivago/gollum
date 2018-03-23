// Copyright 2015-2018 trivago N.V.
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

package format

import (
	"bytes"
	"encoding/json"
	"strconv"
	"sync"
	"time"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tstrings"
)

type jsonReaderState int

const (
	//jsonReadArrayEnd    = jsonReaderState(iota)
	//jsonReadObjectEnd   = jsonReaderState(iota)
	jsonReadObject      = jsonReaderState(iota)
	jsonReadKey         = jsonReaderState(iota)
	jsonReadValue       = jsonReaderState(iota)
	jsonReadArray       = jsonReaderState(iota)
	jsonReadArrayAppend = jsonReaderState(iota)
)

// TextToJSON formatter
//
// This formatter uses a state machine to parse arbitrary text data and
// transform it to JSON.
//
// Parameters
//
// - StartState: Defines the name of the initial state when parsing a message.
// When set to an empty string the first state from the directives array will
// be used.
// By default this parameter is set to "".
//
// - TimestampRead: Defines a time.Parse compatible format string used to read
// time fields when using the "dat" directive.
// By default this parameter is set to "20060102150405".
//
// - TimestampWrite: Defines a time.Format compatible format string used to
// write time fields when using the "dat" directive.
// By default this parameter is set to "2006-01-02 15:04:05 MST".
//
// - UnixTimestampRead: Defines the unix timestamp format expected from fields
// that are parsed using the "dat" directive. Valid valies are "s" for seconds,
// "ms" for milliseconds, or "ns" for nanoseconds. This parameter is ignored
// unless TimestampRead is set to "".
// By default this parameter is set to "".
//
// - Directives: Defines an array of directives used to parse text data.
// Each entry must be of the format: "State:Token:NextState:Flags:Function".
// State denotes the name of the state owning this entry. Multiple entries per
// state are allowed. Token holds a string that triggers a state transition.
// NextState holds the target of the state transition. Flags is an optional
// field and is used to trigger special parser behavior. Flags can be comma
// separated if you need to use more than one.
// Function defines an action that is triggered upon state transition.
// Spaces will be stripped from all fields but Token. If a fields requires a
// colon it has to be escaped with a backslash. Other escape characters
// supported are \n, \r and \t.
// By default this parameter is set to an empty list.
//
// - Directive rules: There are some special cases which will cause the parser
// to do additional actions.
//
// * When writing a value without a key, the state name will become the key.
// * If two keys are written in a row the first key will hold a null value.
// * Writing a key while writing array elements will close the array.
//
// - Directive flags: Flags can modify the parser behavior and can be used to
// store values on a stack across multiple directives.
//
//  - continue: Prepend the token to the next match.
//
//  - append: Append the token to the current match and continue reading.
//
//  - include: Append the token to the current match.
//
//  - push: Push the current state to the stack.
//
//  - pop: Pop the stack and use the returned state if possible.
//
// - Directive actions: Actions are used to write  text read since the last
// transition to the JSON object.
//
//  - key: Write the parsed section as a key.
//
//  - val: Write the parsed section as a value without quotes.
//
//  - esc: Write the parsed section as a escaped string value.
//
//  - dat: Write the parsed section as a timestamp value.
//
//  - arr: Start a new array.
//
//  - obj: Start a new object.
//
//  - end: Close an array or object.
//
//  - arr+val: arr followed by val.
//
//  - arr+esc: arr followed by esc.
//
//  - arr+dat: arr followed by dat.
//
//  - val+end: val followed by end.
//
//  - esc+end: esc followed by end.
//
//  - dat+end: dat followed by end.
//
// Examples
//
// The following example parses JSON data.
//
//  ExampleConsumer:
//    Type: consumer.Console
//    Streams: console
//    Modulators:
//      - format.TextToJSON:
//        Directives:
//          - "findKey   :\":  key       :      :        "
//          - "findKey   :}:             : pop  : end    "
//          - "key       :\":  findVal   :      : key    "
//          - "findVal   :\\:: value     :      :        "
//          - "value     :\":  string    :      :        "
//          - "value     :[:   array     : push : arr    "
//          - "value     :{:   findKey   : push : obj    "
//          - "value     :,:   findKey   :      : val    "
//          - "value     :}:             : pop  : val+end"
//          - "string    :\":  findKey   :      : esc    "
//          - "array     :[:   array     : push : arr    "
//          - "array     :{:   findKey   : push : obj    "
//          - "array     :]:             : pop  : val+end"
//          - "array     :,:   array     :      : val    "
//          - "array     :\":  arrString :      :        "
//          - "arrString :\":  array     :      : esc    "
type TextToJSON struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	message              *bytes.Buffer
	parser               tstrings.TransitionParser
	state                jsonReaderState
	stack                []jsonReaderState
	parseLock            *sync.Mutex
	initState            string `config:"StartState"`
	timeRead             string `config:"TimestampRead"`
	timeWrite            string `config:"TimestampWrite" default:"2006-01-02 15:04:05 MST"`
	timeParse            func(string, string) (time.Time, error)
}

func init() {
	core.TypeRegistry.Register(TextToJSON{})
}

func parseUnix(layout, value string) (time.Time, error) {
	s, ns := int64(0), int64(0)
	switch layout {
	case "s":
		valueInt, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return time.Time{}, err
		}
		s = valueInt
	case "ms":
		valueInt, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return time.Time{}, err
		}
		ns = valueInt * int64(time.Millisecond)
	case "ns":
		valueInt, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return time.Time{}, err
		}
		ns = valueInt
	}
	return time.Unix(s, ns), nil
}

// Configure initializes this formatter with values from a plugin config.
func (format *TextToJSON) Configure(conf core.PluginConfigReader) {
	format.parser = tstrings.NewTransitionParser()
	format.state = jsonReadObject
	format.timeParse = time.Parse
	format.parseLock = new(sync.Mutex)

	unixRead := conf.GetString("UnixTimestampRead", "")
	if format.timeRead == "" {
		if unixRead == "" {
			// Use default when neither are specified
			format.timeRead = "20060102150405"
		} else {
			format.timeRead = unixRead
			format.timeParse = parseUnix
		}
	} else if unixRead != "" {
		format.Logger.Warning("Cannot use both TimestampRead and UnixTimestampRead, defaulting to TimestampRead")
	}

	if !conf.HasValue("Directives") {
		format.Logger.Warning("JSON formatter has no directives setting")
		return // ### return, no directives ###
	}

	directiveStrings := conf.GetStringArray("Directives", []string{})
	if len(directiveStrings) == 0 {
		conf.Errors.Pushf("JSON formatter has no directives")
	} else {
		// Parse directives
		parserFunctions := make(map[string]tstrings.ParsedFunc)
		parserFunctions["key"] = format.readKey
		parserFunctions["val"] = format.readValue
		parserFunctions["esc"] = format.readEscaped
		parserFunctions["dat"] = format.readDate
		parserFunctions["arr"] = format.readArray
		parserFunctions["obj"] = format.readObject
		parserFunctions["end"] = format.readEnd
		parserFunctions["arr+val"] = format.readArrayValue
		parserFunctions["arr+dat"] = format.readArrayDate
		parserFunctions["arr+esc"] = format.readArrayEscaped
		parserFunctions["val+end"] = format.readValueEnd
		parserFunctions["esc+end"] = format.readEscapedEnd
		parserFunctions["dat+end"] = format.readDateEnd

		directives := []tstrings.TransitionDirective{}
		for _, dirString := range directiveStrings {
			directive, err := tstrings.ParseTransitionDirective(dirString, parserFunctions)
			if err != nil {
				conf.Errors.Pushf("%s: %s", err.Error(), dirString)
				continue // ### continue, malformed directive ###
			}

			if format.initState == "" {
				format.initState = directive.State
			}
			directives = append(directives, directive)
		}

		format.parser.AddDirectives(directives)

		// Validate initstate
		initStateValid := false
		for i := 0; i < len(directives) && !initStateValid; i++ {
			initStateValid = directives[i].State == format.initState
		}
		if !initStateValid {
			conf.Errors.Pushf("JSONStartState does not exist in directives")
		}

		for _, dir := range directives {
			nextStateValid := false
			for i := 0; i < len(directives) && !nextStateValid; i++ {
				nextStateValid = dir.NextState == directives[i].State
			}
			if !nextStateValid {
				format.Logger.Warningf("State \"%s\" has a transition to \"%s\" which does not exist in directives", dir.State, dir.NextState)
			}
		}
	}
}

func (format *TextToJSON) writeKey(key []byte) {
	// Make sure we are not in an array anymore
	for format.state == jsonReadArray || format.state == jsonReadArrayAppend {
		format.readEnd(nil, 0)
	}

	// If no value was written, write null
	if format.state > jsonReadKey {
		format.message.WriteString("null")
	}

	// Prepend a comma except after an object has started
	if format.state != jsonReadObject {
		format.message.WriteByte(',')
	}

	format.message.WriteByte('"')
	format.message.Write(key)
	format.message.WriteString(`":`)
}

func (format *TextToJSON) readKey(data []byte, state tstrings.ParserStateID) {
	format.writeKey(data)
	format.state = jsonReadValue
}

func (format *TextToJSON) readValue(data []byte, state tstrings.ParserStateID) {
	trimmedData := bytes.TrimSpace(data)
	if len(trimmedData) == 0 {
		switch format.state {
		default:
			format.state = jsonReadKey
		case jsonReadArray, jsonReadArrayAppend:
		}
		return
	}

	switch format.state {
	default:
		format.writeKey([]byte(format.parser.GetStateName(state)))
		fallthrough

	case jsonReadValue:
		format.message.Write(trimmedData)
		format.state = jsonReadKey

	case jsonReadArray:
		format.message.Write(trimmedData)
		format.state = jsonReadArrayAppend

	case jsonReadArrayAppend:
		format.message.WriteByte(',')
		format.message.Write(trimmedData)
	}
}

func (format *TextToJSON) readEscaped(data []byte, state tstrings.ParserStateID) {
	trimmedData := bytes.TrimSpace(data)
	if len(trimmedData) == 0 {
		switch format.state {
		default:
			format.state = jsonReadKey
		case jsonReadArrayAppend, jsonReadArray:
		}
		return
	}

	encodedData, _ := json.Marshal(string(trimmedData))
	switch format.state {
	default:
		format.writeKey([]byte(format.parser.GetStateName(state)))
		fallthrough

	case jsonReadValue:
		format.message.Write(encodedData)
		format.state = jsonReadKey

	case jsonReadArray:
		format.message.Write(encodedData)
		format.state = jsonReadArrayAppend

	case jsonReadArrayAppend:
		format.message.WriteString(`,`)
		format.message.Write(encodedData)
	}
}

func (format *TextToJSON) readDate(data []byte, state tstrings.ParserStateID) {
	date, _ := format.timeParse(format.timeRead, string(bytes.TrimSpace(data)))
	formattedDate := date.Format(format.timeWrite)
	format.readEscaped([]byte(formattedDate), state)
}

func (format *TextToJSON) readValueEnd(data []byte, state tstrings.ParserStateID) {
	formatState := format.state
	format.readValue(data, state)
	format.state = formatState
	format.readEnd(data, state)
}

func (format *TextToJSON) readEscapedEnd(data []byte, state tstrings.ParserStateID) {
	formatState := format.state
	format.readEscaped(data, state)
	format.state = formatState
	format.readEnd(data, state)
}

func (format *TextToJSON) readDateEnd(data []byte, state tstrings.ParserStateID) {
	formatState := format.state
	format.readDate(data, state)
	format.state = formatState
	format.readEnd(data, state)
}

func (format *TextToJSON) readArrayValue(data []byte, state tstrings.ParserStateID) {
	format.readArray(data, state)
	format.readValue(data, state)
}

func (format *TextToJSON) readArrayEscaped(data []byte, state tstrings.ParserStateID) {
	format.readArray(data, state)
	format.readEscaped(data, state)
}

func (format *TextToJSON) readArrayDate(data []byte, state tstrings.ParserStateID) {
	format.readArray(data, state)
	format.readDate(data, state)
}

func (format *TextToJSON) readArray(data []byte, state tstrings.ParserStateID) {
	switch format.state {
	default:
		format.writeKey([]byte(format.parser.GetStateName(state)))
		fallthrough

	case jsonReadValue, jsonReadArray, jsonReadObject:
		format.message.WriteByte('[')

	case jsonReadArrayAppend:
		format.message.WriteString(",[")
	}
	format.stack = append(format.stack, format.state)
	format.state = jsonReadArray
}

func (format *TextToJSON) readObject(data []byte, state tstrings.ParserStateID) {
	switch format.state {
	default:
		format.writeKey([]byte(format.parser.GetStateName(state)))
		fallthrough

	case jsonReadValue, jsonReadObject, jsonReadArray:
		format.message.WriteByte('{')

	case jsonReadArrayAppend:
		format.message.WriteString(",{")
	}
	format.stack = append(format.stack, format.state)
	format.state = jsonReadObject
}

func (format *TextToJSON) readEnd(data []byte, state tstrings.ParserStateID) {
	stackSize := len(format.stack)

	if stackSize > 0 {
		switch format.state {
		case jsonReadArray, jsonReadArrayAppend:
			format.message.WriteByte(']')
		default:
			format.message.WriteByte('}')
		}
	}

	if stackSize > 1 {
		format.state = format.stack[stackSize-1]
		format.stack = format.stack[:stackSize-1] // Pop the stack
		if format.state == jsonReadArray {
			format.state = jsonReadArrayAppend // just finished the first entry
		}
	} else {
		format.stack = format.stack[:0] // Clear the stack
		format.state = jsonReadKey
	}
}

// ApplyFormatter update message payload
func (format *TextToJSON) ApplyFormatter(msg *core.Message) error {
	// The internal state is not threadsafe so we need to lock here
	format.parseLock.Lock()
	defer format.parseLock.Unlock()

	format.message = bytes.NewBuffer(nil)
	format.state = jsonReadObject

	format.message.WriteString("{")
	remains, state := format.parser.Parse(format.GetAppliedContent(msg), format.initState)

	// Write remains as string value
	if remains != nil {
		format.readEscaped(remains, state)
	}

	// Close any open tags
	if format.message.Len() > 1 {
		for format.state == jsonReadArray || format.state == jsonReadArrayAppend || format.state == jsonReadObject {
			format.readEnd(nil, 0)
		}
	}

	format.message.WriteString("}\n")
	format.SetAppliedContent(msg, bytes.TrimSpace(format.message.Bytes()))

	return nil
}
