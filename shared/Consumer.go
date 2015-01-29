package shared

import "sync"

// ConsumerControl is an enumeration used by the Producer.Control() channel
type ConsumerControl int

const (
	// ConsumerControlStop will cause the consumer to halt and shutdown.
	ConsumerControlStop = ConsumerControl(1)
)

// Consumer is an interface for plugins that recieve data from outside sources
// and generate Message objects from this data.
type Consumer interface {

	// Create a new instance of the concrete plugin class implementing this
	// interface. Expect the instance passed to this function to not be
	// initialized.
	Create(PluginConfig, *BytePool) (Consumer, error)

	// Consume should implement to main loop that fetches messages from a given
	// source and pushes it to the Message channel.
	Consume(*sync.WaitGroup)

	// Control returns write access to this consumer's control channel.
	// See ConsumerControl* constants.
	Control() chan<- ConsumerControl

	// Messages returns read access to the message channel this consumer writes to.
	Messages() <-chan Message
}
