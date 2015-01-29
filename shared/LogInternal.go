package shared

import (
	"fmt"
)

// Internal logging channel
type LogInternal struct {
	Messages chan Message
	Pool     *BytePool
}

// Internal gollum log
var Log = LogInternal{
	Messages: make(chan Message, 1024),
	Pool:     nil,
}

// Write a note to the internal gollum log
func (log LogInternal) Note(args ...interface{}) {
	msg := CreateMessageFromString(log.Pool, fmt.Sprint(args...), LogInternalStreamID)

	select {
	case log.Messages <- msg: // Transfer ownership to channel
	default:
	}
}

// Write a warning to the internal gollum log
func (log LogInternal) Warning(args ...interface{}) {
	msg := CreateMessageFromString(log.Pool, "WARNING: "+fmt.Sprint(args...), LogInternalStreamID)

	select {
	case log.Messages <- msg: // Transfer ownership to channel
	default:
	}
}

// Write an error to the internal gollum log
func (log LogInternal) Error(text string, args ...interface{}) {
	msg := CreateMessageFromString(log.Pool, "ERROR: "+fmt.Sprint(args...), LogInternalStreamID)

	select {
	case log.Messages <- msg: // Transfer ownership to channel
	default:
	}
}
