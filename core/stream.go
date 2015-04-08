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

import "sync/atomic"

var MessageCount = uint32(0)

type Stream interface {
	AddProducer(producers ...Producer)
	Enqueue(msg Message)
}

type StreamBase struct {
	Filter     Filter
	Format     Formatter
	Producers  []Producer
	Distribute func(msg Message)
}

func GetAndResetMessageCount() uint32 {
	return atomic.SwapUint32(&MessageCount, 0)
}

func GetMessageCount() uint32 {
	return MessageCount
}

func (stream *StreamBase) Configure(conf PluginConfig) error {
	plugin, err := NewPluginWithType(conf.GetString("Formatter", "format.Forward"), conf)
	if err != nil {
		return err // ### return, plugin load error ###
	}
	stream.Format = plugin.(Formatter)

	plugin, err = NewPluginWithType(conf.GetString("Filter", "filter.All"), conf)
	if err != nil {
		return err // ### return, plugin load error ###
	}
	stream.Filter = plugin.(Filter)

	stream.Distribute = stream.broadcast
	return nil
}

func (stream *StreamBase) AddProducer(producers ...Producer) {
	for _, prod := range producers {
		for _, inListProd := range stream.Producers {
			if inListProd == prod {
				return // ### return, already in list ###
			}
		}
		stream.Producers = append(stream.Producers, prod)
	}
}

func (stream *StreamBase) broadcast(msg Message) {
	for _, prod := range stream.Producers {
		prod.Enqueue(msg)
	}
}

func (stream *StreamBase) Enqueue(msg Message) {
	atomic.AddUint32(&MessageCount, 1)

	if stream.Filter.Accepts(msg) {
		stream.Format.PrepareMessage(msg)
		msg.Data = stream.Format.Bytes()

		stream.Distribute(msg)
	}
}
