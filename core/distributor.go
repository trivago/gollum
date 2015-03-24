// Copyright 2015 trivago GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package core

// Distributor a distributor plugin can hook on a certain stream and send
// messages to all producers registered to that stream.
type Distributor interface {
	// Distribute sends a message to at least one of the added producers.
	Distribute(msg Message)

	// AddProducer adds a producer to be notified by this distributor when
	// Distribute is called.
	AddProducer(prod Producer)
}

// DistributorBase is the base class for most distributors.
// It implements those function that are most commonly shared between
// distributors.
type DistributorBase struct {
	Producers []Producer
}

// AddProducer adds a producer to the list of producers managed by this
// distributor if it has not yet been added.
func (dist *DistributorBase) AddProducer(prod Producer) {
	for _, inListProd := range dist.Producers {
		if inListProd == prod {
			return // ### return, already in list ###
		}
	}
	dist.Producers = append(dist.Producers, prod)
}
