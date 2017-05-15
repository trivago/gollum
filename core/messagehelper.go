package core

import (
	"strings"
)

// GetAppliedContent is a func() to get message content from payload or meta data
// for later handling by plugins
type GetAppliedContent func(msg *Message) []byte

// GetAppliedContentFunction returns a GetAppliedContent function
func GetAppliedContentFunction(applyTo string) GetAppliedContent {
	parts := strings.Split(applyTo, ":")

	if parts[0] == "meta" {
		return func(msg *Message) []byte {
			return msg.MetaData().GetValue(parts[1], []byte{})
		}
	}

	return func(msg *Message) []byte {
		return msg.Data()
	}
}
