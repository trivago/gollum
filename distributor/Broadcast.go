package distributor

import (
	"github.com/trivago/gollum/shared"
)

// Broadcast distributor plugin
// Configuration example
//
//   - "distributor.Standard":
//     Enable: true
//     Stream: "*"
//
// This consumer does not define any options beside the standard ones.
// Messages are send to all of the producers listening to the given stream.
type Broadcast struct {
}

func init() {
	shared.RuntimeType.Register(Broadcast{})
}

// Configure initializes this distributor with values from a plugin config.
func (dist *Broadcast) Configure(conf shared.PluginConfig) error {
	return nil
}

// Distribute sends the given message to all of the given producers
func (dist *Broadcast) Distribute(message shared.Message, producers []shared.Producer, sendToInactive bool) {
	for _, producer := range producers {
		shared.SingleDistribute(producer, message, sendToInactive)
	}
}
