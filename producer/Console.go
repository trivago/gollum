package producer

import (
	"fmt"
	"github.com/trivago/gollum/shared"
	"os"
	"strings"
	"sync"
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

// Create creates a new producer based on the current console producer.
func (prod Console) Create(conf shared.PluginConfig) (shared.Producer, error) {
	err := prod.configureStandardProducer(conf)
	if err != nil {
		return nil, err
	}

	console := conf.GetString("Console", "stdout")

	switch strings.ToLower(console) {
	default:
		fallthrough
	case "stdout":
		prod.console = os.Stdout
	case "stderr":
		prod.console = os.Stderr
	}

	return prod, nil
}

func (prod Console) printMessage(msg shared.Message) {
	fmt.Fprint(prod.console, prod.format.ToString(msg))
}

func (prod Console) flush() {
	for {
		select {
		case message := <-prod.messages:
			prod.printMessage(message)
		default:
			return
		}
	}
}

// Produce writes to stdout or stderr.
func (prod Console) Produce(threads *sync.WaitGroup) {
	threads.Add(1)
	defer func() {
		prod.flush()
		threads.Done()
	}()

	// Block until one of the channels contains data so we idle when there is
	// nothing to do.

	for {
		select {
		case message := <-prod.messages:
			prod.printMessage(message)

		case command := <-prod.control:
			if command == shared.ProducerControlStop {
				return // ### return, done ###
			}
		}
	}
}
