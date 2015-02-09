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
	shared.RuntimeType.Register(Null{})
}

// Produce writes to a buffer that is dumped to a file.
func (prod Null) Produce(threads *sync.WaitGroup) {
	prod.state.Active = true

	// Block until one of the channels contains data so we idle when there is
	// nothing to do.

	for {
		select {
		case <-prod.messages:
			// Nothing

		case command := <-prod.control:
			if command == shared.ProducerControlStop {
				return // ### return, done ###
			}
		}
	}
}
