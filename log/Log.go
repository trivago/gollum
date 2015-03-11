// Copyright 2015 trivago GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package Log

import (
	"fmt"
	"github.com/trivago/gollum/shared"
	"log"
	"sync/atomic"
)

type logMessages struct {
	queue    chan shared.Message
	sequence uint64
	enqueue  bool
}

type logNull struct {
}

// Verbosity defines an enumeration for log verbosity
type Verbosity byte

var (
	logStreamIDs = []shared.MessageStreamID{shared.LogInternalStreamID}
	logHelper    = logMessages{make(chan shared.Message, 1024), 0, false}
	logDisabled  = logNull{}
)

const (
	// VerbosityError shows only error messages
	VerbosityError = Verbosity(iota)
	// VerbosityWarning shows error and warning messages
	VerbosityWarning = Verbosity(iota)
	// VerbosityNote shows error, warning and note messages
	VerbosityNote = Verbosity(iota)
	// VerbosityDebug shows all messages
	VerbosityDebug = Verbosity(iota)
)

var (
	// Error is a predefined log channel for errors. This log is backed by consumer.Log
	Error *log.Logger

	// Warning is a predefined log channel for warnings. This log is backed by consumer.Log
	Warning *log.Logger

	// Note is a predefined log channel for notes. This log is backed by consumer.Log
	Note *log.Logger

	// Debug is a predefined log channel for debug messages. This log is backed by consumer.Log
	Debug *log.Logger

	// Metric is an alias (reference) to shared.Metric
	Metric = &shared.Metric
)

func init() {
	SetVerbosity(VerbosityError)

	log.SetFlags(log.Lshortfile)
	log.SetOutput(logHelper)
}

// SetVerbosity defines which messages are actually logged.
// See Verbosity.
func SetVerbosity(loglevel Verbosity) {
	Error = log.New(logDisabled, "", 0)
	Warning = log.New(logDisabled, "", 0)
	Note = log.New(logDisabled, "", 0)
	Debug = log.New(logDisabled, "", 0)

	switch loglevel {
	default:
		fallthrough

	case VerbosityDebug:
		Debug = log.New(logHelper, "Debug: ", 0)
		fallthrough

	case VerbosityNote:
		Note = log.New(logHelper, "", 0)
		fallthrough

	case VerbosityWarning:
		Warning = log.New(logHelper, "Warning: ", log.Lshortfile)
		fallthrough

	case VerbosityError:
		Error = log.New(logHelper, "ERROR: ", log.Lshortfile)
	}
}

// Write Drops all messages
func (log logNull) Write(message []byte) (int, error) {
	return len(message), nil
}

// Write pushes messages to the log queue
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

	msg := shared.NewMessage(nil, message, logStreamIDs, atomic.AddUint64(&log.sequence, 1))
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
