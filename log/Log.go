package Log

import (
	"fmt"
	"github.com/trivago/gollum/shared"
	"log"
)

type logMessages struct {
	queue   chan shared.Message
	enqueue bool
}

var logStreamIDs = []shared.MessageStreamID{shared.LogInternalStreamID}
var logHelper = logMessages{make(chan shared.Message, 1024), false}

// Error is a predefined log channel for errors. This log is backed by consumer.Log
var Error *log.Logger

// Error is a predefined log channel for warnings. This log is backed by consumer.Log
var Warning *log.Logger

// Error is a predefined log channel for notes. This log is backed by consumer.Log
var Note *log.Logger

func init() {
	Error = log.New(logHelper, "ERROR: ", log.Lshortfile)
	Warning = log.New(logHelper, "Warning: ", log.Lshortfile)
	Note = log.New(logHelper, "", 0)

	log.SetFlags(log.Lshortfile)
	log.SetOutput(logHelper)
}

func (log logMessages) Write(message []byte) (int, error) {
	length := len(message)
	if length == 0 {
		return 0, nil
	}

	if !log.enqueue {
		fmt.Print(string(message))
		return length, nil
	}

	if message[length-1] == '\n' {
		message = message[:length-1]
	}

	msg := shared.NewMessageFromSlice(message, logStreamIDs)
	log.queue <- msg

	return length, nil
}

// EnqueueMessages can switch between pushing messages to stderr or the internal
// log stream (false, true). By default messages are written to stderr.
func EnqueueMessages(enable bool) {
	logHelper.enqueue = enable
}

// Messages returns a read only access to queued log messages
func Messages() <-chan shared.Message {
	return logHelper.queue
}
