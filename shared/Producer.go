package shared

import "sync"

// ProducerControl is an enumeration used by the Producer.Control() channel
type ProducerControl int

const (
	// ProducerControlStop will cause the producer to halt and shutdown.
	ProducerControlStop = ProducerControl(1)
)

// Producer is an interface for plugins that pass Message objects to other
// services, files or storages.
type Producer interface {

	// Create a new instance of the concrete plugin class implementing this
	// interface. Expect the instance passed to this function to not be
	// initialized.
	Create(PluginConfig) (Producer, error)

	// Main loop that passes messages from the message channel to some other
	// service like the console.
	Produce(*sync.WaitGroup)

	// Returns true if the message is allowed to be send to this producer.
	Accepts(message Message) bool

	// Returns write access to this producer's control channel.
	// See ProducerControl* constants.
	Control() chan<- ProducerControl

	// Returns write access to the message channel this producer reads from.
	Messages() chan<- Message
}
