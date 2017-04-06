// Copyright 2015-2016 trivago GmbH
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

import (
	"github.com/trivago/tgo/tlog"
	"time"
)

type SimpleStream struct {
	id         string
	modulators ModulatorArray
	Producers  []Producer
	Timeout    *time.Duration
	streamID   MessageStreamID
	Log        tlog.LogScope
}

// Configure sets up all values required by SimpleStream.
func (stream *SimpleStream) Configure(conf PluginConfigReader) error {
	stream.id = conf.GetID()
	stream.Log = conf.GetLogScope()
	stream.Timeout = nil
	stream.streamID = conf.GetStreamID("Stream", GetStreamID(conf.GetID()))
	stream.modulators = conf.GetModulatorArray("Modulators", stream.Log, ModulatorArray{})

	if stream.streamID == WildcardStreamID {
		stream.Log.Note.Print("A wildcard stream configuration only affects the wildcard stream, not all streams")
	}

	if conf.HasValue("TimeoutMs") {
		timeout := time.Duration(conf.GetInt("TimeoutMs", 0)) * time.Millisecond
		stream.Timeout = &timeout
	}

	return conf.Errors.OrNil()
}

// GetID returns the ID of this stream
func (stream *SimpleStream) GetID() string {
	return stream.id
}

// StreamID returns the id of the stream this plugin is bound to.
func (stream *SimpleStream) StreamID() MessageStreamID {
	return stream.streamID
}

// AddProducer adds all producers to the list of known producers.
// Duplicates will be filtered.
func (stream *SimpleStream) AddProducer(producers ...Producer) {
	for _, prod := range producers {
		for _, inListProd := range stream.Producers {
			if inListProd == prod {
				return // ### return, already in list ###
			}
		}
		stream.Producers = append(stream.Producers, prod)
	}
}

// GetProducers returns the producers bound to this stream
func (stream *SimpleStream) GetProducers() []Producer {
	return stream.Producers
}

// Modulate calls all modulators in their order of definition
func (stream *SimpleStream) Modulate(msg *Message) ModulateResult {
	return stream.modulators.Modulate(msg)
}
