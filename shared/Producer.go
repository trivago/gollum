package shared

const (
	// Event to pass to Producer.Control.
	// Will cause the producer to halt and shutdown.
	ProducerControlStop = 1
)

// Producers are plugins that generate messages, i.e. read them from some source
// and provide them for other plugins to consume.
type Producer interface {

	// Create a new instance of the concrete plugin class implementing this
	// interface. Expect the instance passed to this function to not be
	// initialized.
	Create(PluginConfig) (Producer, error)

	// Main loop that passes messages from the message channel to some other
	// service like the console.
	Produce()

	// Returns write access to this producer's control channel.
	// See ProducerControl* constants.
	Control() chan<- int

	// Returns write access to the message channel this producer reads from.
	Messages() chan<- Message
}
