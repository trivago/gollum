package producer

import (
	"fmt"
	"gollum/shared"
)

type Console struct {
	messages chan string
	control  chan int
}

var ConsoleClassID = shared.Plugin.Register(Console{})

func (prod Console) Create(conf shared.PluginConfig) (shared.Producer, error) {
	prod.messages = make(chan string)
	prod.control = make(chan int)
	return prod, nil
}

func (prod Console) Produce() {
	for {
		select {
		case message := <-prod.messages:
			fmt.Println(message)

		case command := <-prod.control:
			if command == shared.ProducerControlStop {
				fmt.Println("Console producer recieved stop")
				return // ### return ###
			}
		default:
			// Don't block
		}
	}
}

func (prod Console) Control() chan<- int {
	return prod.control
}

func (prod Console) Messages() chan<- string {
	return prod.messages
}
