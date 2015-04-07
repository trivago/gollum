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

import (
	"github.com/trivago/gollum/core/log"
	"hash/fnv"
)

type StreamRegistry struct {
	streams map[MessageStreamID]Stream
	name    map[MessageStreamID]string
}

var StreamTypes = StreamRegistry{
	streams: make(map[MessageStreamID]Stream),
	name:    make(map[MessageStreamID]string),
}

// GetStreamID returns the integer representation of a given stream name.
func GetStreamID(stream string) MessageStreamID {
	hash := fnv.New64a()
	hash.Write([]byte(stream))
	streamID := MessageStreamID(hash.Sum64())

	StreamTypes.name[streamID] = stream
	return streamID
}

func (registry *StreamRegistry) GetStreamName(streamID MessageStreamID) string {
	if name, exists := registry.name[streamID]; exists {
		return name // ### return, found ###
	}
	return ""
}

func (registry *StreamRegistry) GetStreamByName(name string) Stream {
	streamID := GetStreamID(name)
	return registry.GetStream(streamID)
}

func (registry *StreamRegistry) GetStream(id MessageStreamID) Stream {
	stream, exists := registry.streams[id]
	if !exists {
		return nil
	}
	return stream
}

func (registry *StreamRegistry) Register(stream Stream, streamID MessageStreamID) {
	if _, exists := registry.streams[streamID]; exists {
		Log.Warning.Printf("%T attaches to an already occupied stream (%s)", stream, registry.GetStreamName(streamID))
	}
	registry.streams[streamID] = stream
}
