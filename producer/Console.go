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
//   - "producer.Console":
//     Enable: true
//     Console: "stderr"
//
// Console may either be "stdout" or "stderr"
type Console struct {
	shared.ProducerBase
	console *os.File
}

func init() {
	shared.RuntimeType.Register(Console{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *Console) Configure(conf shared.PluginConfig) error {
	err := prod.ProducerBase.Configure(conf)
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
	prod.Formatter().PrepareMessage(msg)
	fmt.Fprint(prod.console, prod.Formatter().String())
}

func (prod Console) flush() {
	for prod.NextNonBlocking(prod.printMessage) {
	}
}

// Produce writes to stdout or stderr.
func (prod Console) Produce(threads *sync.WaitGroup) {
	defer func() {
		prod.flush()
		prod.MarkAsDone()
	}()

	prod.DefaultControlLoop(threads, prod.printMessage, nil)
}
