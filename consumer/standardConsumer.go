package consumer

import (
	"github.com/trivago/gollum/shared"
)

// Consumer base class
// All consumers support a common subset of configuration options:
//
// - "consumer.Something":
//   Enable: true
//   Buffer: 1024
//   Stream:
//      - "error"
//      - "default"
//
// Enable switches the consumer on or off. By default this value is set to true.
// Buffer set the size of the channel used to communicate messages. By default
// this value is set to 1024
// Stream contains either a single string or a list of strings defining the
// message channels this consumer will produce. By default this is set to "*"
// which means only producers set to consume "all streams" will get these
// messages.
type standardConsumer struct {
	messages chan shared.Message
	control  chan shared.ConsumerControl
	stream   []shared.MessageStreamID
	pool     *shared.BytePool
}

func (cons *standardConsumer) configureStandardConsumer(conf shared.PluginConfig, pool *shared.BytePool) error {
	cons.messages = make(chan shared.Message, conf.Channel)
	cons.control = make(chan shared.ConsumerControl, 1)
	cons.stream = make([]shared.MessageStreamID, len(conf.Stream))
	cons.pool = pool

	for i, stream := range conf.Stream {
		cons.stream[i] = shared.GetStreamID(stream)
	}

	return nil
}

func (cons standardConsumer) postMessage(text string) {
	msg := shared.CreateMessageFromString(cons.pool, text, shared.WildcardStreamID)

	for _, stream := range cons.stream {
		msg.StreamID = stream
		msg.Data.Acquire() // Acquire ownership for channel
		cons.messages <- msg
	}

	msg.Data.Release() // Release ownership for this function
}

func (cons standardConsumer) postMessageFromSlice(data []byte) {
	msg := shared.CreateMessage(cons.pool, data, 0)

	for _, stream := range cons.stream {
		msg.StreamID = stream
		msg.Data.Acquire() // Acquire ownership for channel
		cons.messages <- msg
	}

	msg.Data.Release() // Release ownership for this function
}

func (cons standardConsumer) Control() chan<- shared.ConsumerControl {
	return cons.control
}

func (cons standardConsumer) Messages() <-chan shared.Message {
	return cons.messages
}
