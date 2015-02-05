package shared

import (
	"log"
)

var logStreamIDs = []MessageStreamID{LogInternalStreamID}

type logs struct {
	messages chan Message
	Errors   *log.Logger
	Warnings *log.Logger
	Notes    *log.Logger
}

// Log is the "namespace" containing all gollum loggers
var Log logs

func init() {
	Log = logs{messages: make(chan Message, 1024)}

	Log.Errors = log.New(Log, "ERROR: ", log.Lshortfile)
	Log.Warnings = log.New(Log, "Warning: ", log.Lshortfile)
	Log.Notes = log.New(Log, "", 0)

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

	msg := CreateMessageFromSlice(message, logStreamIDs)
	log.messages <- msg

	return length, nil
}

// LogMessages returns read-only access to the log messages currently queued
func (log logs) Messages() <-chan Message {
	return log.messages
}

// Fatal is a shortcut to Log.Errors.Fatal
func (log logs) Fatal(args ...interface{}) {
	log.Errors.Fatal(args)
}

// Fatalf is a shortcut to Log.Errors.Fatalf
func (log logs) Fatalf(format string, args ...interface{}) {
	log.Errors.Fatalf(format, args)
}

// Panic is a shortcut to Log.Errors.Panic
func (log logs) Panic(args ...interface{}) {
	log.Errors.Panic(args)
}

// Panicf is a shortcut to Log.Errors.Panicf
func (log logs) Panicf(format string, args ...interface{}) {
	log.Errors.Panicf(format, args)
}

// Error is a shortcut to Log.Errors.Print
func (log logs) Error(args ...interface{}) {
	log.Errors.Print(args)
}

// Errorf is a shortcut to Log.Errors.Printf
func (log logs) Errorf(format string, args ...interface{}) {
	log.Errors.Printf(format, args)
}

// Warning is a shortcut to Log.Warning.Print
func (log logs) Warning(args ...interface{}) {
	log.Warnings.Print(args)
}

// Warningf is a shortcut to Log.Warning.Printf
func (log logs) Warningf(format string, args ...interface{}) {
	log.Warnings.Printf(format, args)
}

// Note is a shortcut to Log.Note.Print
func (log logs) Note(args ...interface{}) {
	log.Notes.Print(args)
}

// Notef is a shortcut to Log.Note.Printf
func (log logs) Notef(format string, args ...interface{}) {
	log.Notes.Printf(format, args)
}
