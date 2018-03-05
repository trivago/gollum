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

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"hash/fnv"
	"time"
)

type messageTracer struct {
	msg      *Message
	pluginID string
}

// codebeat:disable[TOO_MANY_IVARS]
type messageDump struct {
	Step            string
	PluginID        string
	Payload         string
	OriginalPayload string
	Stream          string
	PrevStream      string
	OrigStream      string
	Metadata        map[string]string
	Source          string
	Timestamp       time.Time
	FingerprintID   uint32
}

// codebeat:enable[TOO_MANY_IVARS]

type messageTracerSource struct {
}

// IsActive returns true if the source can produce messages
func (mts messageTracerSource) IsActive() bool {
	return true
}

// IsBlocked returns true if the source cannot produce messages
func (mts messageTracerSource) IsBlocked() bool {
	return false
}

// GetID returns the pluginID of the message source
func (mts messageTracerSource) GetID() string {
	return "core.MessageTracer"
}

// MessageTrace provide the MessageTrace() function. By default this function do nothing.
var MessageTrace = func(msg *Message, pluginID string, comment string) {}

// ActivateMessageTrace set a MessageTrace function to dump out the message trace
func ActivateMessageTrace() {
	MessageTrace = func(msg *Message, pluginID string, comment string) {
		mt := messageTracer{
			msg:      msg,
			pluginID: pluginID,
		}

		mt.Dump(comment)
	}
}

// DeactivateMessageTrace set a MessageTrace function to default
// This method is necessary for unit testing
func DeactivateMessageTrace() {
	MessageTrace = func(msg *Message, pluginID string, comment string) {}
}

// Dump creates a messageDump struct for the message trace
func (mt *messageTracer) Dump(comment string) {
	if mt.msg.streamID == LogInternalStreamID || mt.msg.streamID == TraceInternalStreamID {
		return
	}

	var err error
	msgDump := mt.newMessageDump(mt.msg, comment)

	if StreamRegistry.IsStreamRegistered(TraceInternalStreamID) {
		err = mt.routeToTraceStream(msgDump)

	} else {
		err = mt.printMessageTrace(msgDump)
	}

	if err != nil {
		logrus.Error(err)
	}
}

func (mt *messageTracer) printMessageTrace(dump messageDump) error {
	jsonData, err := json.MarshalIndent(dump, "", "\t")
	if err == nil {
		_, err = fmt.Println("\n", string(jsonData))
	}

	return err
}

func (mt *messageTracer) routeToTraceStream(dump messageDump) error {
	jsonData, err := json.Marshal(dump)
	if err != nil {
		return err
	}

	traceMsg := NewMessage(messageTracerSource{}, jsonData, nil, TraceInternalStreamID)
	traceRouter := StreamRegistry.GetRouterOrFallback(TraceInternalStreamID)

	return traceRouter.Enqueue(traceMsg)
}

func (mt *messageTracer) newMessageDump(msg *Message, comment string) messageDump {
	dump := messageDump{}
	// set general trace info
	dump.Step = comment
	dump.PluginID = mt.pluginID

	// prepare message payloads data
	dump.Payload = msg.String()
	if msg.orig != nil {
		dump.OriginalPayload = string(msg.orig.payload)
	}

	// prepare message streams data
	dump.Stream = mt.getStreamName(msg.streamID)
	dump.PrevStream = mt.getStreamName(msg.prevStreamID)
	dump.OrigStream = mt.getStreamName(msg.origStreamID)

	// prepare meta data
	if metadata := msg.TryGetMetadata(); metadata != nil {
		dump.Metadata = map[string]string{}
		for k, v := range metadata {
			dump.Metadata[k] = string(v)
		}
	}

	// set source data
	if msg.source != nil {
		dump.Source = msg.source.GetID()
	}

	//  set timestamp
	dump.Timestamp = msg.timestamp

	dump.FingerprintID = mt.createFingerPrintID(&dump)

	return dump
}

func (mt *messageTracer) createFingerPrintID(dump *messageDump) uint32 {
	hash := fnv.New32a()

	hash.Write([]byte(dump.Source))
	hash.Write([]byte(dump.Timestamp.String()))

	return hash.Sum32()
}

// get stream name and translate InvalidStreamID ("") to InvalidStream
func (mt *messageTracer) getStreamName(streamID MessageStreamID) string {
	if streamID == InvalidStreamID {
		return "InvalidStream"
	}

	return StreamRegistry.GetStreamName(streamID)
}
