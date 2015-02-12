package trivago

import (
	"bytes"
	"encoding/json"
	"github.com/trivago/gollum/shared"
	"time"
)

type errlTransition struct {
	character rune
	state     int
}

const (
	errlStateServer  = 0
	errlStateSearch  = 1
	errlStateSection = 2
	errlStateDate    = 3
	errlStateStatus  = 4
	errlStateClient  = 5
	errlStateRemote  = 6
)

var errlSectionDoneTransition = shared.NewTransition("]", errlStateSearch, shared.ParserFlagDone)

var errlTransitions = [][]shared.Transition{
	/* server  */ {shared.NewTransition("[", errlStateDate, shared.ParserFlagDone)},
	/* search  */ {shared.NewTransition("[", errlStateSection, shared.ParserFlagStart)},
	/* section */ {shared.NewTransition("remote", errlStateRemote, shared.ParserFlagStart),
		shared.NewTransition("client", errlStateClient, shared.ParserFlagStart),
		shared.NewTransition("error", errlStateStatus, shared.ParserFlagNop),
		shared.NewTransition("warning", errlStateStatus, shared.ParserFlagNop)},
	/* date    */ {errlSectionDoneTransition},
	/* status  */ {errlSectionDoneTransition},
	/* client  */ {errlSectionDoneTransition},
	/* remote  */ {errlSectionDoneTransition},
}

type ErrorLogFormatter struct {
	parser  shared.Parser
	message *bytes.Buffer
}

func init() {
	shared.RuntimeType.Register(ErrorLogFormatter{})
}

func (format *ErrorLogFormatter) Configure(conf shared.PluginConfig) error {
	format.parser = shared.NewParser(errlTransitions)
	return nil
}

func (format *ErrorLogFormatter) writeField(name string, data []byte, first *bool) {
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

func (format *ErrorLogFormatter) PrepareMessage(msg shared.Message) {
	sections := format.parser.Parse([]byte(msg.Data))
	isFirst := true
	format.message = bytes.NewBufferString("{")
	for _, section := range sections {
		switch section.State {
		case errlStateServer:
			format.writeField("server", section.Data, &isFirst)

		case errlStateDate:
			timeStamp, _ := time.Parse("Mon Jan 2 15:04:05 2006", string(section.Data))
			format.writeField("@timestamp", []byte(timeStamp.Format(time.RFC3339)), &isFirst)

		case errlStateStatus:
			format.writeField("status", section.Data, &isFirst)

		case errlStateClient:
			format.writeField("client", section.Data, &isFirst)

		case errlStateRemote:
			format.writeField("remote", section.Data, &isFirst)

		default:
			format.writeField("message", section.Data, &isFirst)
		}
	}
	format.message.WriteString("}")
}

func (format *ErrorLogFormatter) GetLength() int {
	return format.message.Len()
}

func (format *ErrorLogFormatter) String() string {
	return format.message.String()
}

func (format *ErrorLogFormatter) CopyTo(dest []byte) int {
	return copy(dest, format.message.Bytes())
}
