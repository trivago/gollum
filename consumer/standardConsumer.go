package consumer

import (
	"gollum/shared"
)

type standardConsumer struct {
	messages chan shared.Message
	control  chan int
	response chan int
}

func (cons *standardConsumer) configureStandardConsumer(conf shared.PluginConfig) error {
	cons.messages = make(chan shared.Message)
	cons.control = make(chan int, 1)
	cons.response = make(chan int, 1)
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
