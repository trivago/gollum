package trivago

import (
	"bytes"
	"github.com/trivago/gollum/shared"
	"time"
)

type errlTransition struct {
	character rune
	state     int
}

const (
	errlStateServer = iota
	errlStateSearch
	errlStateSection
	errlStateDate
	errlStateStatus
	errlStateClient
	errlStateRemote
)

var errlStateNames = []string{
	"server",
	"message",
	"message",
	"@timestamp",
	"status",
	"client",
	"remote",
}

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
	JSONLogFormatter
}

func init() {
	shared.RuntimeType.Register(ErrorLogFormatter{})
}

func (format *ErrorLogFormatter) Configure(conf shared.PluginConfig) error {
	format.parser = shared.NewParser(errlTransitions)
	return nil
}

func (format *ErrorLogFormatter) PrepareMessage(msg shared.Message) {
	sections := format.parser.Parse([]byte(msg.Data))
	isFirst := true
	format.message = bytes.NewBufferString("{")

	for _, section := range sections {
		switch section.State {
		case errlStateDate:
			timeStamp, _ := time.Parse("Mon Jan 2 15:04:05 2006", string(bytes.TrimSpace(section.Data)))
			format.writeField("@timestamp", []byte(timeStamp.Format(time.RFC3339)), &isFirst)

		default:
			format.writeField(errlStateNames[section.State], section.Data, &isFirst)
		}
	}

	format.message.WriteString("}")
}
