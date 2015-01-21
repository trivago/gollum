package shared

import (
	"fmt"
	"time"
)

const (
	LogInternalStream = "_GOLLUM_"
)

type LogInternal struct {
	Messages chan Message
}

var Log = LogInternal{make(chan Message, 1024)}

func (log LogInternal) Note(args ...interface{}) {
	msg := Message{
		Text:      fmt.Sprint(args...),
		Stream:    LogInternalStream,
		Timestamp: time.Now(),
	}

	select {
	case log.Messages <- msg:
	default:
	}
}

func (log LogInternal) Warning(args ...interface{}) {
	msg := Message{
		Text:      "WARNING:" + fmt.Sprint(args),
		Stream:    LogInternalStream,
		Timestamp: time.Now(),
	}

	select {
	case log.Messages <- msg:
	default:
	}
}

func (log LogInternal) Error(text string, args ...interface{}) {
	msg := Message{
		Text:      "ERROR: " + fmt.Sprint(args...),
		Stream:    LogInternalStream,
		Timestamp: time.Now(),
	}

	select {
	case log.Messages <- msg:
	default:
	}
}
