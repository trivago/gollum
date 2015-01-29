package consumer

import (
	"bufio"
	"github.com/trivago/gollum/shared"
	"os"
	"sync"
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

func (cons Console) Create(conf shared.PluginConfig, pool *shared.BytePool) (shared.Consumer, error) {
	err := cons.configureStandardConsumer(conf, pool)
	return cons, err
}

func (cons Console) readFrom(stream *os.File) {
	var err error
	var message string

	reader := bufio.NewReader(stream)

	for {

		// TODO: This call blocks and prevents this go routine from shutting
		// 		 down properly

		message, err = reader.ReadString('\n')
		if err != nil {
			return // ### return, stream error ###
		}

		cons.postMessage(message[:len(message)-1])
	}
}

func (cons Console) Consume(threads *sync.WaitGroup) {
	go cons.readFrom(os.Stdin)

	// Wait for control statements

	for {
		command := <-cons.control
		if command == shared.ConsumerControlStop {
			return // ### return ###
		}
	}
}
