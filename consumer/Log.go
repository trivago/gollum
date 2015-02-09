package consumer

import (
	"github.com/trivago/gollum/shared"
	"sync"
)

// Log consumer plugin
// This is an internal plugin to route go log messages into gollum
type Log struct {
	control chan shared.ConsumerControl
}

// Configure initializes this consumer with values from a plugin config.
func (cons *Log) Configure(conf shared.PluginConfig) error {
	cons.control = make(chan shared.ConsumerControl, 1)
	return nil
}

// IsActive always returns true for this consumer
func (cons Log) IsActive() bool {
	return true
}

// Control returns a handle to the control channel
func (cons Log) Control() chan<- shared.ConsumerControl {
	return cons.control
}

// Messages reroutes Log.Messages()
func (cons Log) Messages() <-chan shared.Message {
	return shared.Log.Messages()
}

// Consume starts listening for control statements
func (cons Log) Consume(threads *sync.WaitGroup) {
	// Wait for control statements
	for {
		command := <-cons.control
		if command == shared.ConsumerControlStop {
			return // ### return ###
		}
	}
}
