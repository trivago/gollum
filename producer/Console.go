package producer

import (
	"fmt"
	"gollum/shared"
	"os"
	"strings"
)

// Console producer plugin
// Configuration example
//
// - "producer.Console":
//   Enable: true
//   Console: "stderr"
//
// Console may either be "stdout" or "stderr"
type Console struct {
	standardProducer
	console *os.File
}

func init() {
	shared.Plugin.Register(Console{})
}

func (prod Console) Create(conf shared.PluginConfig) (shared.Producer, error) {
	err := prod.configureStandardProducer(conf)
	if err != nil {
		return nil, err
	}

	console, consoleSet := conf.Settings["Console"]

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
			fmt.Fprintln(prod.console, message.Format(prod.forward))

		case command := <-prod.control:
			if command == shared.ProducerControlStop {
				return // ### return, done ###
			}
		default:
			// Don't block
		}
	}
}
