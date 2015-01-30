package consumer

import (
	"github.com/trivago/gollum/shared"
	"io"
	"os"
	"sync"
)

const (
	consoleBufferGrowSize = 1024
)

// Console consumer plugin
// Configuration example
//
// - "consumer.Console":
//   Enable: true
//
// This consumer does not define any options beside the standard ones.
type Console struct {
	standardConsumer
}

func init() {
	shared.Plugin.Register(Console{})
}

// Create creates a new consumer based on the current console consumer.
func (cons Console) Create(conf shared.PluginConfig, pool *shared.BytePool) (shared.Consumer, error) {
	err := cons.configureStandardConsumer(conf, pool)
	return cons, err
}

func (cons *Console) readFrom(stream io.Reader, threads *sync.WaitGroup) {
	buffer := shared.CreateBufferedReader(consoleBufferGrowSize, cons.postMessageFromSlice)

	for {
		err := buffer.Read(os.Stdin, "\n")
		if err != nil {
			shared.Log.Error("Error reading stdin: ", err)
		}
	}
}

// Consume listens to stdin.
func (cons Console) Consume(threads *sync.WaitGroup) {
	go cons.readFrom(os.Stdin, threads)

	// Wait for control statements

	for {
		command := <-cons.control
		if command == shared.ConsumerControlStop {
			return // ### return ###
		}
	}
}
