package distributor

import (
	"github.com/trivago/gollum/shared"
)

// RoundRobin distributor plugin
// Configuration example
//
// - "distributor.RoundRobin":
//   Enable: true
//   Stream: "*"
//
// This consumer does not define any options beside the standard ones.
// Messages are send to one of the producers listening to the given stream.
// The target producer changes after each send.
type RoundRobin struct {
	index map[shared.MessageStreamID]int
}

func init() {
	shared.RuntimeType.Register(RoundRobin{})
}

// Configure initializes this distributor with values from a plugin config.
func (dist *RoundRobin) Configure(conf shared.PluginConfig) error {
	dist.index = make(map[shared.MessageStreamID]int)
	return nil
}

// Distribute sends the given message to one of the given producers in a round
// robin fashion.
func (dist *RoundRobin) Distribute(message shared.Message, producers []shared.Producer, sendToInactive bool) {

	// As we might listen to different streams we have to keep the index for
	// each stream separately
	index, isSet := dist.index[message.PinnedStream]
	if !isSet {
		index = 0
	} else {
		index %= len(producers)
	}

	shared.SingleDistribute(producers[index], message, sendToInactive)
	dist.index[message.PinnedStream] = index + 1
}
