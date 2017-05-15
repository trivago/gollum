package core

import (
	"github.com/trivago/tgo/tlog"
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

// RouteOriginal restores the original message and routes it by using the
// currently set stream.
func RouteOriginal(msg *Message) error {
	return RouteOriginalByRouter(msg, msg.GetRouter())
}

// RouteOriginalByRouter restores the original message and routes it by using a
// a given router.
func RouteOriginalByRouter(msg *Message, router Router) error {
	err := Route(msg.CloneOriginal(), router)
	if err != nil {
		tlog.Error.Printf("Routing error: Can't route message by '%T': %s", router, err.Error())
	}
	return err
}

// DiscardMessage count the discard statistic and stop msg handling
// after a Discard() call stop further message handling
func DiscardMessage(msg *Message) {
	CountDiscardedMessage()
}
