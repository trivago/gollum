package producer

import (
	"fmt"
	"gollum/shared"
	"os"
	"regexp"
	"strings"
)

type Console struct {
	messages chan shared.Message
	control  chan int
	response chan int
	channel  string
	filter   *regexp.Regexp
	console  *os.File
}

var ConsoleClassID = shared.Plugin.Register(Console{})

func (prod Console) Create(conf shared.PluginConfig) (shared.Producer, error) {
	console, consoleSet := conf.Settings["Console"]
	channel, channelSet := conf.Settings["Channel"]
	filter, filterSet := conf.Settings["Filter"]

	if !channelSet {
		prod.channel = ""
	} else {
		prod.channel = channel.(string)
	}

	if !filterSet {
		prod.filter = nil
	} else {
		var err error
		prod.filter, err = regexp.Compile(filter.(string))
		if err != nil {
			return nil, err
		}
	}

	if !consoleSet {
		prod.console = os.Stdout
	} else {
		switch strings.ToLower(console.(string)) {
		default:
			fallthrough
		case "stdout":
			prod.console = os.Stdout
		case "stderr":
			prod.console = os.Stderr
		}
	}

	prod.messages = make(chan shared.Message)
	prod.control = make(chan int, 1)
	prod.response = make(chan int, 1)

	return prod, nil
}

func (prod Console) Produce() {
	defer func() {
		prod.response <- shared.ProducerControlResponseDone
	}()

	for {
		select {
		case message := <-prod.messages:
			if prod.channel == "" {
				fmt.Fprintln(prod.console, message.Format())
			} else {
				fmt.Fprintf(prod.console, "%s %s: %s\n", message.GetDateString(), prod.channel, message.Text)
			}

		case command := <-prod.control:
			if command == shared.ProducerControlStop {
				fmt.Println("Console producer recieved stop")
				return // ### return, done ###
			}
		default:
			// Don't block
		}
	}
}

func (prod Console) Accepts(message shared.Message) bool {
	if prod.filter == nil {
		return true // ### return, pass everything ###
	}

	return prod.filter.MatchString(message.Text)
}

func (prod Console) Control() chan<- int {
	return prod.control
}

func (prod Console) ControlResponse() <-chan int {
	return prod.response
}

func (prod Console) Messages() chan<- shared.Message {
	return prod.messages
}
