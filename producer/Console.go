package producer

import (
	"fmt"
	"gollum/shared"
	"os"
	"strings"
)

type Console struct {
	standardProducer
	channel string
	console *os.File
}

var ConsoleClassID = shared.Plugin.Register(Console{})

func (prod Console) Create(conf shared.PluginConfig) (shared.Producer, error) {

	err := prod.configureStandardProducer(conf)
	if err != nil {
		return nil, err
	}

	console, consoleSet := conf.Settings["Console"]
	channel, channelSet := conf.Settings["Channel"]

	if !channelSet {
		prod.channel = ""
	} else {
		prod.channel = channel.(string)
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
				//fmt.Println("Console producer recieved stop")
				return // ### return, done ###
			}
		default:
			// Don't block
		}
	}
}
