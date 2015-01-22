package consumer

import (
	"gollum/shared"
	"time"
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
	control  chan int
	response chan int
	stream   []string
}

func (cons *standardConsumer) configureStandardConsumer(conf shared.PluginConfig) error {
	cons.messages = make(chan shared.Message, conf.Channel)
	cons.control = make(chan int, 1)
	cons.response = make(chan int, 1)
	cons.stream = conf.Stream

	return nil
}

func (cons standardConsumer) postMessage(text string) {
	msg := shared.Message{
		Text:      text,
		Timestamp: time.Now(),
	}
	for _, stream := range cons.stream {
		msg.Stream = stream
		cons.messages <- msg
	}
}

func (cons standardConsumer) Control() chan<- int {
	return cons.control
}

func (cons standardConsumer) ControlResponse() <-chan int {
	return cons.response
}

func (cons standardConsumer) Messages() <-chan shared.Message {
	return cons.messages
}
