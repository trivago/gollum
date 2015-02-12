package trivago

import (
	"bytes"
	"github.com/trivago/gollum/shared"
	"time"
)

type acclTransition struct {
	character rune
	state     int
}

const (
	acclStateServerIP = iota
	acclStateServerName
	acclStateClientIP
	acclStateClientIP2
	acclStateClientIP3
	acclStateTimestamp
	acclStateMethod
	acclStateRequest
	acclStateProtocol
	acclStateResultCode
	acclStateRequestSize
	acclStateResponseSize
	acclStateCommand
	acclStateFlags
	acclStateReferrer
	acclStateSession
	acclStateAgent
)

var acclStateNames = []string{
	"serverIP",
	"serverName",
	"clientIP",
	"clientIP2",
	"clientIP3",
	"@timestamp",
	"method",
	"request",
	"protocol",
	"resultCode",
	"requestSize",
	"responseSize",
	"command",
	"flags",
	"referrer",
	"session",
	"agent",
}

var acclTransitions = [][]shared.Transition{
	/* serverIP   */ {shared.NewTransition(" ", acclStateServerName, shared.ParserFlagDone)},
	/* serverName */ {shared.NewTransition(" ", acclStateClientIP, shared.ParserFlagDone)},
	/* clientIP   */ {shared.NewTransition(" ", acclStateClientIP3, shared.ParserFlagDone), shared.NewTransition(",", acclStateClientIP2, shared.ParserFlagDone)},
	/* clientIP2  */ {shared.NewTransition(" ", acclStateClientIP3, shared.ParserFlagDone)},
	/* clientIP3  */ {shared.NewTransition(" ", acclStateTimestamp, shared.ParserFlagDone)},
	/* timestamp  */ {shared.NewTransition("\"", acclStateMethod, shared.ParserFlagDone)},
	/* method     */ {shared.NewTransition(" ", acclStateRequest, shared.ParserFlagDone)},
	/* request    */ {shared.NewTransition(" ", acclStateProtocol, shared.ParserFlagDone)},
	/* protocol   */ {shared.NewTransition("\" ", acclStateResultCode, shared.ParserFlagDone)},
	/* resultCode */ {shared.NewTransition(" ", acclStateRequestSize, shared.ParserFlagDone)},
	/* number1    */ {shared.NewTransition(" ", acclStateResponseSize, shared.ParserFlagDone)},
	/* size       */ {shared.NewTransition(" ", acclStateCommand, shared.ParserFlagDone)},
	/* command    */ {shared.NewTransition(" ", acclStateFlags, shared.ParserFlagDone)},
	/* flags      */ {shared.NewTransition("\"", acclStateReferrer, shared.ParserFlagDone)},
	/* referrer   */ {shared.NewTransition("\" \"", acclStateSession, shared.ParserFlagDone)},
	/* session    */ {shared.NewTransition("\" \"", acclStateAgent, shared.ParserFlagDone)},
	/* agent      */ {shared.NewTransition("\"", shared.ParserStateStop, shared.ParserFlagDone)},
}

type AccessLogFormatter struct {
	JSONLogFormatter
}

func init() {
	shared.RuntimeType.Register(AccessLogFormatter{})
}

func (format *AccessLogFormatter) Configure(conf shared.PluginConfig) error {
	format.parser = shared.NewParser(acclTransitions)
	return nil
}

func (format *AccessLogFormatter) PrepareMessage(msg shared.Message) {
	sections := format.parser.Parse([]byte(msg.Data))
	isFirst := true
	format.message = bytes.NewBufferString("{")

	for _, section := range sections {
		switch section.State {
		case acclStateTimestamp:
			timeStamp, _ := time.Parse("20060102150405", string(bytes.TrimSpace(section.Data)))
			format.writeField("@timestamp", []byte(timeStamp.Format(time.RFC3339)), &isFirst)

		default:
			format.writeField(acclStateNames[section.State], section.Data, &isFirst)
		}
	}

	format.message.WriteString("}")
}
