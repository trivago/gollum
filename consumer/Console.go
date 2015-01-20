package consumer

import (
	"bufio"
	"fmt"
	"gollum/shared"
	"os"
	"time"
)

type Console struct {
	messages chan shared.Message
	control  chan int
	response chan int
}

var ConsoleClassID = shared.Plugin.Register(Console{})

func (cons Console) Create(conf shared.PluginConfig) (shared.Consumer, error) {
	cons.messages = make(chan shared.Message)
	cons.control = make(chan int, 1)
	cons.response = make(chan int, 1)
	return cons, nil
}

func (cons Console) readFrom(stream *os.File) {
	var err error
	var message string

	reader := bufio.NewReader(stream)

	for {

		// TODO: This call blocks and prevents this go routine from shutting
		// 		 down properly

		message, err = reader.ReadString('\n')
		if err != nil {
			return // ### return, stream error ###
		}

		postMessage := shared.Message{
			Text:      message[:len(message)-1],
			Timestamp: time.Now(),
		}

		// Async positing of messages

		select {
		case cons.messages <- postMessage:
		default:
		}
	}
}

func (cons Console) Consume() {

	go cons.readFrom(os.Stdin)
	defer func() {
		cons.response <- shared.ConsumerControlResponseDone
	}()

	// Wait for control statements

	for {
		command := <-cons.control
		if command == shared.ConsumerControlStop {
			fmt.Println("Console consumer recieved stop")
			return // ### return ###
		}
	}
}

func (cons Console) Control() chan<- int {
	return cons.control
}

func (cons Console) ControlResponse() <-chan int {
	return cons.response
}

func (cons Console) Messages() <-chan shared.Message {
	return cons.messages
}
