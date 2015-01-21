package consumer

import (
	"gollum/shared"
)

// Consumer base class
// All consumers support a common subset of configuration options:
//
// - "consumer.Something":
//   Enable: true
//   Buffer: 1024
//   Forward: false
//   Stream:
//      - "error"
//      - "default"
//
// Enable switches the consumer on or off. By default this value is set to false.
// Buffer set the size of the channel used to communicate messages. By default
// this value is set to 1024
// Stream contains either a single string or a list of strings defining the
// message channels this consumer will produce. By default this is set to "*"
// which means only producers set to consume "all streams" will get these
// messages.
// If forward is set to true, the message will be passed as-is, so date and
// channel will not be added. The default value is false.
type standardConsumer struct {
	messages chan shared.Message
	control  chan int
	response chan int
	stream   []string
	forward  bool
}

func (cons *standardConsumer) configureStandardConsumer(conf shared.PluginConfig) error {
	cons.messages = make(chan shared.Message, conf.Buffer)
	cons.control = make(chan int, 1)
	cons.response = make(chan int, 1)
	cons.stream = conf.Stream
	cons.forward = false

	forward, forwardSet := conf.Settings["Forward"]
	if forwardSet {
		cons.forward = forward.(bool)
	}

	return nil
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
