// Copyright 2015-2018 trivago N.V.
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

// MessageStreamID is the "compiled name" of a stream
type MessageStreamID uint64

const (
	// InvalidStream is used for invalid stream handling and maps to ""
	InvalidStream = ""
	// LogInternalStream is the name of the internal message channel (logs)
	LogInternalStream = "_GOLLUM_"
	// TraceInternalStream is the name of the internal trace channel (-tm flag)
	TraceInternalStream = "_TRACE_"
	// WildcardStream is the name of the "all routers" channel
	WildcardStream = "*"
)

var (
	// InvalidStreamID denotes an invalid stream for function returing stream IDs
	InvalidStreamID = GetStreamID(InvalidStream)
	// LogInternalStreamID is the ID of the "_GOLLUM_" stream
	LogInternalStreamID = GetStreamID(LogInternalStream)
	// WildcardStreamID is the ID of the "*" stream
	WildcardStreamID = GetStreamID(WildcardStream)
	// TraceInternalStreamID is the ID of the "_TRACE_" stream
	TraceInternalStreamID = GetStreamID(TraceInternalStream)
)
