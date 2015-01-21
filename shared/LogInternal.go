package shared

import (
	"fmt"
	"time"
)

const (
	LogInternalStream = "_GOLLUM_"
)

// Internal logging channel
type LogInternal struct {
	Messages chan Message
}

// Internal gollum log
var Log = LogInternal{make(chan Message, 1024)}

// Write a note to the internal gollum log
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

// Write a warning to the internal gollum log
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

// Write an error to the internal gollum log
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
