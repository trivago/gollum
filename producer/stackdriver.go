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

package producer

import (
	"context"
	"encoding/json"
	"sync"

	"cloud.google.com/go/logging"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tcontainer"
)

// Stackdriver producer plugin
//
// The Stackdriver producer forwards messages to stackdriver logging.
// The payload of each message will be passed as log message. This should ideally
// be a JSON encoded string.
//
// Parameters
//
// - ProjectID: The google cloud project id storing the logs. This parameter
// is required to be set to a non-empty string.
//
// - Payload: The name of the metadata field to store as payload. This can be a
// string or a subtree, which will be encoded as JSON. If no value is set, the
// message payload will be passed as string.
// By default this is setting is set to "".
//
// - LogName: Defines a mapping between stream name and log name. If not set,
// the stream name will be used as a log name. By default no mapping is set.
//
// - Labels: An array of metadata keys, that should be extracted from each message
// and used as a label. If a key is not available for a single message, the label
// will still be set with an empty string as value.
// By default this is set to an empty array.
//
// - Severity: If set, this denotes a metadata field containing a valid severity
// string. Valid, case sensitive strings are "Default", "Debug", "Info", "Notice",
// "Warning", "Error", "Critical", "Alert" and "Emergency". If a value fails to be
// parsed "Default" is used. If the metadata field is not existing or "" is used,
// the value of DefaultSeverity is used.
// By default this is setting is set to "".
//
// - DefaultSeverity: The severity to use if no severity is set or parsing of the
// Severity metadata field failed. By default this is set to "Default".
//
// Examples
//
//   StackdriverOut:
//     Type: producer.Stackdriver
//     Streams: mylogs
//     ProjectID: my-gcp-project-id
//     Payload: "message"
//     Severity: "severity"
//     DefaultSeverity: "Error"
// 	   LogName:
//       "mylogs": "errorlog"
//     Labels:
//       - "hostname"
//       - "service"
//
type Stackdriver struct {
	core.BufferedProducer `gollumdoc:"embed_type"`
	ProjectID             string `config:"ProjectID"`
	LogName               map[core.MessageStreamID]string
	Labels                []string `config:"Labels"`
	Payload               string   `config:"Payload"`
	Severity              string   `config:"Severity"`
	DefaultSeverity       string   `config:"DefaultSeverity" default:"Default"`

	client   *logging.Client
	logger   map[core.MessageStreamID]*logging.Logger
	severity logging.Severity
}

func init() {
	core.TypeRegistry.Register(Stackdriver{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *Stackdriver) Configure(conf core.PluginConfigReader) {
	prod.logger = make(map[core.MessageStreamID]*logging.Logger)
	prod.LogName = conf.GetStreamMap("LogName", core.LogInternalStream)

	if len(prod.ProjectID) == 0 {
		conf.Errors.Pushf("A ProjectID must be provided.")
	}

	prod.severity = logging.ParseSeverity(prod.DefaultSeverity)
}

func (prod *Stackdriver) parseSeverity(metadata tcontainer.MarshalMap) logging.Severity {
	if len(prod.Severity) == 0 {
		return prod.severity
	}

	if metadata == nil {
		return prod.severity
	}

	value, ok := metadata.Value(prod.Severity)
	if !ok {
		return prod.severity
	}

	severityString := core.ConvertToString(value)
	if len(severityString) == 0 {
		return prod.severity
	}

	return logging.ParseSeverity(severityString)
}

func (prod *Stackdriver) parseLabels(metadata tcontainer.MarshalMap) map[string]string {
	if len(prod.Labels) == 0 {
		return nil
	}

	if metadata == nil {
		return nil
	}

	labels := make(map[string]string)

	for _, key := range prod.Labels {
		if value, ok := metadata.Value(key); ok {
			labels[key] = core.ConvertToString(value)
		} else {
			labels[key] = ""
		}
	}

	return labels
}

func marshalMapToProtoStruct(m tcontainer.MarshalMap) *structpb.Struct {
	fields := map[string]*structpb.Value{}
	for k, v := range m {
		fields[k] = jsonValueToStructValue(v)
	}
	return &structpb.Struct{Fields: fields}
}

func jsonMapToProtoStruct(m map[string]interface{}) *structpb.Struct {
	fields := map[string]*structpb.Value{}
	for k, v := range m {
		if pbValue := jsonValueToStructValue(v); pbValue != nil {
			fields[k] = pbValue
		}
	}
	return &structpb.Struct{Fields: fields}
}

func jsonValueToStructValue(v interface{}) *structpb.Value {
	switch x := v.(type) {
	case bool:
		return &structpb.Value{Kind: &structpb.Value_BoolValue{BoolValue: x}}
	case float64:
		return &structpb.Value{Kind: &structpb.Value_NumberValue{NumberValue: x}}
	case string:
		return &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: x}}
	case nil:
		return &structpb.Value{Kind: &structpb.Value_NullValue{}}
	case map[string]interface{}:
		return &structpb.Value{Kind: &structpb.Value_StructValue{StructValue: jsonMapToProtoStruct(x)}}
	case []interface{}:
		var vals []*structpb.Value
		for _, e := range x {
			vals = append(vals, jsonValueToStructValue(e))
		}
		return &structpb.Value{Kind: &structpb.Value_ListValue{ListValue: &structpb.ListValue{Values: vals}}}
	default:
		return nil
	}
}

func (prod *Stackdriver) getPayload(msg *core.Message, metadata tcontainer.MarshalMap) interface{} {
	if len(prod.Payload) == 0 {
		return json.RawMessage(msg.GetPayload())
	}

	if metadata == nil {
		return ""
	}

	value, ok := metadata.Value(prod.Payload)
	if !ok {
		return ""
	}

	switch v := value.(type) {
	case string:
		if v[0] == '{' {
			return json.RawMessage(v)
		}
		return v

	case []byte:
		if v[0] == '{' {
			return json.RawMessage(v)
		}
		return v

	case tcontainer.MarshalMap:
		return marshalMapToProtoStruct(v)

	case map[string]interface{}:
		return jsonMapToProtoStruct(v)

	default:
		return v
	}
}

func (prod *Stackdriver) printMessage(msg *core.Message) {
	logger, ok := prod.logger[msg.GetStreamID()]
	if !ok {
		streamID := msg.GetStreamID()
		logName, ok := prod.LogName[streamID]
		if !ok {
			logName = streamID.GetName()
		}
		logger = prod.client.Logger(logName)
		prod.logger[msg.GetStreamID()] = logger
	}

	metadata := msg.TryGetMetadata()

	logger.Log(logging.Entry{
		Timestamp: msg.GetCreationTime(),
		Payload:   prod.getPayload(msg, metadata),
		Severity:  prod.parseSeverity(metadata),
		Labels:    prod.parseLabels(metadata),
	})
}

func (prod *Stackdriver) createClient() (err error) {
	prod.client, err = logging.NewClient(context.Background(), prod.ProjectID)
	return err
}

// Produce writes to stdout or stderr.
func (prod *Stackdriver) Produce(workers *sync.WaitGroup) {
	if err := prod.createClient(); err != nil {
		prod.Logger.WithError(err).Errorf("failed to create stackdriver client")
		return
	}

	defer prod.client.Close()
	defer prod.WorkerDone()

	prod.AddMainWorker(workers)
	prod.MessageControlLoop(prod.printMessage)
}
