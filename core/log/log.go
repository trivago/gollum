// Copyright 2015-2016 trivago GmbH
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
	"io"
	"log"
)

// Verbosity defines an enumeration for log verbosity
type Verbosity byte

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
	Error = log.New(logDisabled, "", 0)

	// Warning is a predefined log channel for warnings. This log is backed by consumer.Log
	Warning = log.New(logDisabled, "", 0)

	// Note is a predefined log channel for notes. This log is backed by consumer.Log
	Note = log.New(logDisabled, "", 0)

	// Debug is a predefined log channel for debug messages. This log is backed by consumer.Log
	Debug = log.New(logDisabled, "", 0)

	logEnabled  = logReferrer{new(logCache)}
	logDisabled = logNull{}
)

func init() {
	log.SetFlags(0)
	log.SetOutput(logEnabled)
	SetVerbosity(VerbosityError)
}

// SetVerbosity defines the type of messages to be processed.
// High level verobosities contain lower levels, i.e. log level warning will
// contain error messages, too.
func SetVerbosity(loglevel Verbosity) {
	Error = log.New(logDisabled, "", 0)
	Warning = log.New(logDisabled, "", 0)
	Note = log.New(logDisabled, "", 0)
	Debug = log.New(logDisabled, "", 0)

	switch loglevel {
	default:
		fallthrough

	case VerbosityDebug:
		Debug = log.New(&logEnabled, "Debug: ", 0)
		fallthrough

	case VerbosityNote:
		Note = log.New(&logEnabled, "", 0)
		fallthrough

	case VerbosityWarning:
		Warning = log.New(&logEnabled, "Warning: ", log.Lshortfile)
		fallthrough

	case VerbosityError:
		Error = log.New(&logEnabled, "ERROR: ", log.Lshortfile)
	}
}

// SetWriter forces (enabled) logs to be written to the given writer.
func SetWriter(writer io.Writer) {
	oldWriter := logEnabled.writer
	logEnabled.writer = writer
	if cache, isCache := oldWriter.(*logCache); isCache {
		cache.flush(logEnabled)
	}
}
