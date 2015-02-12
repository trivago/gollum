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

func (format *JSONLogFormatter) writeField(name string, data []byte, first *bool) {
	if !*first {
		format.message.WriteString(",\"")
	} else {
		format.message.WriteString("\"")
		*first = false
	}
	format.message.WriteString(name)
	format.message.WriteString("\":\"")
	json.HTMLEscape(format.message, bytes.TrimSpace(data))
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
