package shared

import (
	"fmt"
)

// LogInternal manages the internal gollum logging channel
type LogInternal struct {
	Messages chan Message
	streams  []MessageStreamID
}

// Log is the global instance of gollums logging channel
var Log = LogInternal{
	Messages: make(chan Message, 1024),
	streams:  []MessageStreamID{LogInternalStreamID},
}

// Note writes a message to the internal gollum log
func (log LogInternal) Note(args ...interface{}) {
	msg := CreateMessage(fmt.Sprint(args...), log.streams)

	select {
	case log.Messages <- msg: // Transfer ownership to channel
	default:
	}
}

// Warning writes a warning message to the internal gollum log
func (log LogInternal) Warning(args ...interface{}) {
	msg := CreateMessage("WARNING: "+fmt.Sprint(args...), log.streams)

	select {
	case log.Messages <- msg: // Transfer ownership to channel
	default:
	}
}

// Error writes an error message to the internal gollum log
func (log LogInternal) Error(text string, args ...interface{}) {
	msg := CreateMessage("ERROR: "+fmt.Sprint(args...), log.streams)

	select {
	case log.Messages <- msg: // Transfer ownership to channel
	default:
	}
}
