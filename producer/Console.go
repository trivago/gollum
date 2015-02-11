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
	shared.RuntimeType.Register(Console{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *Console) Configure(conf shared.PluginConfig) error {
	err := prod.standardProducer.Configure(conf)
	if err != nil {
		return err
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

	return nil
}

func (prod Console) printMessage(msg shared.Message) {
	fmt.Fprint(prod.console, prod.format.String(msg))
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
	defer func() {
		prod.flush()
		prod.markAsDone()
	}()

	prod.defaultControlLoop(threads, prod.printMessage)
}
