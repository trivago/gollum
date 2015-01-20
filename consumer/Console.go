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
}

var ConsoleClassID = shared.Plugin.Register(Console{})

func (cons Console) Create(conf shared.PluginConfig) (shared.Consumer, error) {
	cons.messages = make(chan shared.Message)
	cons.control = make(chan int)
	return cons, nil
}

func (cons Console) Consume() {
	reader := bufio.NewReader(os.Stdin)

	for {
		select {
		case command := <-cons.control:
			if command == shared.ConsumerControlStop {
				fmt.Println("Console consumer recieved stop")
				return // ### return ###
			}
		default:
			// Don't block
		}

		message, _ := reader.ReadString('\n')
		cons.messages <- shared.Message{message[:len(message)-1], time.Now()}
	}
}

func (cons Console) Control() chan<- int {
	return cons.control
}

func (cons Console) Messages() <-chan shared.Message {
	return cons.messages
}
