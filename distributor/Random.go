package distributor

import (
	"github.com/trivago/gollum/shared"
	"math/rand"
)

// Random distributor plugin
// Configuration example
//
//   - "distributor.Random":
//     Enable: true
//     Stream: "*"
//
// This consumer does not define any options beside the standard ones.
// Messages are send to a random producer in the set of the producers listening
// to the given stream.
type Random struct {
}

func init() {
	shared.RuntimeType.Register(Random{})
}

// Configure initializes this distributor with values from a plugin config.
func (dist *Random) Configure(conf shared.PluginConfig) error {
	return nil
}

// Distribute sends the given message to one random producer in the set of
// given producers.
func (dist *Random) Distribute(message shared.Message, producers []shared.Producer, sendToInactive bool) {
	index := rand.Intn(len(producers))
	shared.SingleDistribute(producers[index], message, sendToInactive)
}
