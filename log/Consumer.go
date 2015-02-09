package Log

import (
	"github.com/trivago/gollum/shared"
	"sync"
)

// Consumer consumer plugin
// This is an internal plugin to route go log messages into gollum
type Consumer struct {
	control chan shared.ConsumerControl
}

// Configure initializes this consumer with values from a plugin config.
func (cons *Consumer) Configure(conf shared.PluginConfig) error {
	cons.control = make(chan shared.ConsumerControl, 1)
	return nil
}

// IsActive always returns true for this consumer
func (cons Consumer) IsActive() bool {
	return true
}

// Control returns a handle to the control channel
func (cons Consumer) Control() chan<- shared.ConsumerControl {
	return cons.control
}

// Messages reroutes Log.Messages()
func (cons Consumer) Messages() <-chan shared.Message {
	return Messages()
}

// Consume starts listening for control statements
func (cons Consumer) Consume(threads *sync.WaitGroup) {
	// Wait for control statements
	for {
		command := <-cons.control
		if command == shared.ConsumerControlStop {
			return // ### return ###
		}
	}
}
