package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"os"
)

// fallbackLogDevice defines the fallback destination for when _GOLLUM_ is unavailable / not used
var fallbackLogDevice = os.Stderr

// logrusHookBuffer is our single instance of LogrusHookBuffer
var logrusHookBuffer = LogrusHookBuffer{}

// LogrusHookBuffer implements logrus.Hook and is used to pools log messages during startup
// when the desired log destination (_GOLLUM_ stream or fallbackLogDevice) is not yet available
// or known. During normal operation it relays messages to targetHook and/or targetWriter.
type LogrusHookBuffer struct {
	targetHook   logrus.Hook
	targetWriter io.Writer
	buffer       []*logrus.Entry
}

// Levels and Fire() implement the logrus.Hook interface
func (lhb *LogrusHookBuffer) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire and Levels() implement the logrus.Hook interface.
func (lhb *LogrusHookBuffer) Fire(logrusEntry *logrus.Entry) error {
	if lhb.targetHook == nil && lhb.targetWriter == nil {
		// Store entry in buffer and return
		lhb.buffer = append(lhb.buffer, logrusEntry)
		return nil
	}

	// Handle entry directly
	return lhb.relayEntry(logrusEntry)
}

// SetTargetHook sets the logrus hook to whose .Fire() method messages should be relayed
func (lhb *LogrusHookBuffer) SetTargetHook(hook logrus.Hook) {
	lhb.targetHook = hook
}

// SetTargetWriter sets the io.Writer where messages should be written
func (lhb *LogrusHookBuffer) SetTargetWriter(writer io.Writer) {
	lhb.targetWriter = writer
}

// Purge sends stored messages to targetHook and/or targetWriter and empties the buffer.
func (lhb *LogrusHookBuffer) Purge() {
	// Relay all stored entries
	for _, entry := range lhb.buffer {
		lhb.relayEntry(entry)
	}

	// Empty the buffer
	lhb.buffer = []*logrus.Entry{}
}

// relayEntry relays one entry to the targetHook and/or writes it to targetWriter.
func (lhb *LogrusHookBuffer) relayEntry(entry *logrus.Entry) error {
	// Relay entry to final hook
	if lhb.targetHook != nil {
		err := lhb.targetHook.Fire(entry)
		if err != nil {
			return err
		}
	}

	// Relay entry to Writer
	if lhb.targetWriter != nil {
		serialized, err := entry.Logger.Formatter.Format(entry)
		if err != nil {
			_ = fmt.Errorf("failed to serialize log entry: %s", err.Error())
			return err
		}

		_, err = lhb.targetWriter.Write(serialized)
		if err != nil {
			_ = fmt.Errorf("failed to write log entry %s: %s", serialized, err.Error())
			return err
		}
	}

	// Success
	return nil
}
