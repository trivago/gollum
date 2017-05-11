package core

import (
	"strings"
	"github.com/trivago/tgo/tlog"
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

// DropMessage restore the original message and "drops" they the active message stream router
func DropMessage(msg *Message) error {
	return DropMessageByRouter(msg, msg.GetRouter())
}

// DropMessageByRouter restore the original message and "drops" they to specific router
func DropMessageByRouter(msg *Message, router Router) error {
	CountDroppedMessage()
	err := Route(msg.CloneOriginal(), router)
	if err != nil {
		tlog.Error.Printf("DropMessage error: Can't route message by '%T': %s",router, err.Error())

	}

	return err
}

// DiscardMessage count the discard statistic and stop msg handling
// after a Discard() call stop further message handling
func DiscardMessage(msg *Message) {
	CountDiscardedMessage()
}
