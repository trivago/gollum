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
	errlStateSearch = iota
	errlStateSection
	errlStateServer
	errlStateDate
	errlStateStatus
	errlStateRemote
	errlStateClient
	errlStateMessage
)

var errlStateNames = []string{
	"message",
	"message",
	"server",
	"@timestamp",
	"status",
	"remote",
	"client",
	"message",
}

var errlSectionDoneTransition = transWrite("]", errlStateSearch)

var errlTransitions = [][]shared.Transition{
	/* search  */ {transSkip("[", errlStateSection)},
	/* section */ {transSkip("remote ", errlStateRemote),
		transSkip("client ", errlStateClient),
		transIgnore("error", errlStateStatus),
		transIgnore("warning", errlStateStatus)},

	/* server  */ {transWrite(" [", errlStateDate)},
	/* date    */ {errlSectionDoneTransition},
	/* status  */ {errlSectionDoneTransition},
	/* remote  */ {errlSectionDoneTransition},
	/* client  */ {transWrite("] ", errlStateMessage)},
	/* message */ {},
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
	sections := format.parser.Parse(msg.Data, errlStateServer)
	isFirst := true
	format.message = bytes.NewBufferString("{")

	for _, section := range sections {
		switch section.State {
		case errlStateDate:
			timeStamp, _ := time.Parse("Mon Jan 2 15:04:05 2006", string(section.Data))
			format.writeField("@timestamp", []byte(timeStamp.Format(time.RFC3339)), &isFirst)

		default:
			format.writeField(errlStateNames[section.State], section.Data, &isFirst)
		}
	}

	format.message.WriteString("}")
}
