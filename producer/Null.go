package producer

import (
	"github.com/trivago/gollum/shared"
	"sync"
)

// Null producer plugin
// Configuration example
//
//   - "producer.Null":
//     Enable: true
//
// This producer does nothing.
// It shows the most basic producer who can exist.
// In combination with the consumer "Profiler" this can act as a performance test
type Null struct {
	shared.ProducerBase
}

func init() {
	shared.RuntimeType.Register(Null{})
}

func (prod Null) testFormatter(msg shared.Message) {
	prod.Formatter().PrepareMessage(msg)
	prod.Formatter().String()
}

// Produce writes to a buffer that is dumped to a file.
func (prod Null) Produce(threads *sync.WaitGroup) {
	defer prod.MarkAsDone()

	prod.DefaultControlLoop(threads, prod.testFormatter, nil)
}
