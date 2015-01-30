package producer

import (
	"github.com/trivago/gollum/shared"
	"sync"
)

// Null producer plugin
// Configuration example
//
// - "producer.Null":
//   Enable: true
//
// This producer does nothing.
// It shows the most basic producer who can exist.
// In combination with the consumer "Profiler" this can act as a performance test
type Null struct {
	standardProducer
}

func init() {
	shared.Plugin.Register(Null{})
}

// Create creates a new producer based on the current Null producer.
func (prod Null) Create(conf shared.PluginConfig) (shared.Producer, error) {
	err := prod.configureStandardProducer(conf)
	if err != nil {
		return nil, err
	}

	return prod, nil
}

// Produce writes to a buffer that is dumped to a file.
func (prod Null) Produce(threads *sync.WaitGroup) {
	// Block until one of the channels contains data so we idle when there is
	// nothing to do.

	for {
		select {
		case message := <-prod.messages:
			message.Release()

		case command := <-prod.control:
			if command == shared.ProducerControlStop {
				return // ### return, done ###
			}
		}
	}
}
