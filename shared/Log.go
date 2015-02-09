package shared

import (
	"log"
)

var logStreamIDs = []MessageStreamID{LogInternalStreamID}

type logs struct {
	messages chan Message
	Error    *log.Logger
	Warning  *log.Logger
	Note     *log.Logger
}

// Log is the "namespace" containing all gollum loggers
var Log logs

func init() {
	Log = logs{messages: make(chan Message, 1024)}

	Log.Error = log.New(Log, "ERROR: ", log.Lshortfile)
	Log.Warning = log.New(Log, "Warning: ", log.Lshortfile)
	Log.Note = log.New(Log, "", 0)

	log.SetFlags(log.Lshortfile)
	log.SetOutput(Log)
}

func (log logs) Write(message []byte) (int, error) {
	length := len(message)
	if length == 0 {
		return 0, nil
	}

	if message[length-1] == '\n' {
		message = message[:length-1]
	}

	msg := NewMessageFromSlice(message, logStreamIDs)
	log.messages <- msg

	return length, nil
}

func (log logs) Messages() <-chan Message {
	return log.messages
}
