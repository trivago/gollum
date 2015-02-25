package shared

// Distributor a distributor plugin can hook on a certain stream and send
// messages to all producers registered to that stream.
type Distributor interface {

	// Distribute is meant to send the passed message to at least one of the
	// given producers. This can be used for analyzing messages as well as load
	// distribution. The sendToInactive flag is set to false if inactive
	// producers are to be ignored (may happen during shutdown)
	Distribute(message Message, producers []Producer, sendToInactive bool)
}

// SingleDistribute is the default function for distributing a message to a
// single producer. This method should be used at the core of each Distributor.
func SingleDistribute(prod Producer, message Message, sendToInactive bool) {
	if (prod.IsActive() || sendToInactive) && prod.Accepts(message) {
		PostMessage(prod.Messages(), message, prod.GetTimeout())
	}
}
