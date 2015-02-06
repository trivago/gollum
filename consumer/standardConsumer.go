package consumer

import (
	"github.com/trivago/gollum/shared"
)

// Consumer base class
// All consumers support a common subset of configuration options:
//
// - "consumer.Something":
//   Enable: true
//   Channel: 1024
//   Stream:
//      - "error"
//      - "default"
//
// Enable switches the consumer on or off. By default this value is set to true.
//
// Channel sets the size of the channel used to communicate messages. By default
// this value is set to 1024
//
// Stream contains either a single string or a list of strings defining the
// message channels this consumer will produce. By default this is set to "*"
// which means only producers set to consume "all streams" will get these
// messages.
type standardConsumer struct {
	messages chan shared.Message
	control  chan shared.ConsumerControl
	streams  []shared.MessageStreamID
}

func (cons *standardConsumer) configureStandardConsumer(conf shared.PluginConfig) error {
	cons.messages = make(chan shared.Message, conf.Channel)
	cons.control = make(chan shared.ConsumerControl, 1)
	cons.streams = make([]shared.MessageStreamID, len(conf.Stream))

	for i, stream := range conf.Stream {
		cons.streams[i] = shared.GetStreamID(stream)
	}

	return nil
}

// postMessage sends a message text to all configured streams.
// This method blocks of the message queue is full.
func (cons standardConsumer) postMessage(text string) {
	msg := shared.CreateMessage(text, cons.streams)
	cons.messages <- msg
}

// postMessageFromSlice sends a buffered message to all configured streams.
// This method blocks of the message queue is full.
func (cons standardConsumer) postMessageFromSlice(data []byte) {
	msg := shared.CreateMessageFromSlice(data, cons.streams)
	cons.messages <- msg
}

func (cons standardConsumer) Control() chan<- shared.ConsumerControl {
	return cons.control
}

func (cons standardConsumer) Messages() <-chan shared.Message {
	return cons.messages
}
