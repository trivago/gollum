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
	filters    []Filter
	formatters []Formatter
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

	if stream.streamID == WildcardStreamID {
		stream.Log.Note.Print("A wildcard stream configuration only affects the wildcard stream, not all streams")
	}

	if conf.HasValue("TimeoutMs") {
		timeout := time.Duration(conf.GetInt("TimeoutMs", 0)) * time.Millisecond
		stream.Timeout = &timeout
	}

	formatPlugins := conf.GetPluginArray("Formatters", []Plugin{})
	for _, plugin := range formatPlugins {
		formatter, isFormatter := plugin.(Formatter)
		if !isFormatter {
			conf.Errors.Pushf("Plugin is not a valid formatter")
		} else {
			formatter.SetLogScope(stream.Log)
			stream.formatters = append(stream.formatters, formatter)
		}
	}

	filterPlugins := conf.GetPluginArray("Filters", []Plugin{})
	for _, plugin := range filterPlugins {
		filter, isFilter := plugin.(Filter)
		if !isFilter {
			conf.Errors.Pushf("Plugin is not a valid filter")
		} else {
			filter.SetLogScope(stream.Log)
			stream.filters = append(stream.filters, filter)
		}
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

// Accepts calls applys all filters to the given message and returns true when
// all filters pass.
func (stream *SimpleStream) Accepts(msg *Message) bool {
	for _, filter := range stream.filters {
		if !filter.Accepts(msg) {
			filter.Drop(msg)
			CountFilteredMessage()
			return false // ### return, false if one filter failed ###
		}
	}
	return true
}

// Format calls all formatters in their order of definition
func (stream *SimpleStream) Format(msg *Message) {
	for _, formatter := range stream.formatters {
		formatter.Format(msg)
	}
}
