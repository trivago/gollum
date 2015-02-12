package shared

import "sync"

// ProducerControl is an enumeration used by the Producer.Control() channel
type ProducerControl int

const (
	// ProducerControlStop will cause the producer to halt and shutdown.
	ProducerControlStop = ProducerControl(1)

	// ProducerControlRoll notifies the consumer about a log rotation or
	// revalidation/reconnect of the write target
	ProducerControlRoll = ProducerControl(2)
)

// Producer is an interface for plugins that pass Message objects to other
// services, files or storages.
type Producer interface {
	// Produce should implement the main loop that passes messages from the
	// message channel to some other service like the console.
	Produce(*sync.WaitGroup)

	// IsActive returns true if the producer is ready to accept new data.
	IsActive() bool

	// Accepts returns true if the message is allowed to be send to this producer.
	Accepts(message Message) bool

	// Control returns write access to this producer's control channel.
	// See ProducerControl* constants.
	Control() chan<- ProducerControl

	// Messages returns write access to the message channel this producer reads from.
	Messages() chan<- Message
}
