package producer

import (
	"fmt"
	"gollum/shared"
	"os"
	"strings"
)

type Console struct {
	messages chan shared.Message
	control  chan int
	channel  string
	console  *os.File
}

var ConsoleClassID = shared.Plugin.Register(Console{})

func (prod Console) Create(conf shared.PluginConfig) (shared.Producer, error) {
	console, consoleSet := conf.Settings["Console"]
	channel, channelSet := conf.Settings["Channel"]
	var consoleFile *os.File

	if !consoleSet {
		console = "stdout"
	}

	if !channelSet {
		channel = console
	}

	switch strings.ToLower(console.(string)) {
	default:
		fallthrough
	case "stdout":
		consoleFile = os.Stdout
	case "stderr":
		consoleFile = os.Stderr
	}

	prod.messages = make(chan shared.Message)
	prod.control = make(chan int)
	prod.channel = channel.(string)
	prod.console = consoleFile

	return prod, nil
}

func (prod Console) Produce() {
	for {
		select {
		case message := <-prod.messages:
			fmt.Fprintln(prod.console, prod.channel, message.Format())

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

func (prod Console) Messages() chan<- shared.Message {
	return prod.messages
}
