package trivago

import (
	"bytes"
	"encoding/json"
	"github.com/trivago/gollum/shared"
)

type JSONLogFormatter struct {
	parser  shared.Parser
	message *bytes.Buffer
}

func zapInvalidChars(data []byte) {
	for idx, char := range data {
		switch char {
		case '\r':
			fallthrough
		case '\n':
			data[idx] = '\t'
		case '"':
			data[idx] = '\''
		default:
		}
	}
}

func (format *JSONLogFormatter) writeField(name string, data []byte, first *bool) {
	if !*first {
		format.message.WriteString(",\"")
	} else {
		format.message.WriteString("\"")
		*first = false
	}

	zapInvalidChars(data)
	format.message.WriteString(name)
	format.message.WriteString("\":\"")
	json.HTMLEscape(format.message, data)
	format.message.WriteString("\"")
}

func (format *JSONLogFormatter) GetLength() int {
	return format.message.Len()
}

func (format *JSONLogFormatter) String() string {
	return format.message.String()
}

func (format *JSONLogFormatter) CopyTo(dest []byte) int {
	return copy(dest, format.message.Bytes())
}

func transWrite(token string, state int) shared.Transition {
	return shared.NewTransition(token, state, shared.ParserFlagDone)
}

func transSkip(token string, state int) shared.Transition {
	return shared.NewTransition(token, state, shared.ParserFlagSkip)
}

func transIgnore(token string, state int) shared.Transition {
	return shared.NewTransition(token, state, shared.ParserFlagIgnore)
}
