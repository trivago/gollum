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
	"sync"

	"cloud.google.com/go/logging"
	"github.com/trivago/gollum/core"
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
// parsed, or the severity field is not available in metadata, "Default" is assumed.
// By default this is set to "" which defaults to "Default" for all messages.
//
// Examples
//
//   StackdriverOut:
//     Type: producer.Stackdriver
//     Streams: mylogs
//     ProjectID: my-gcp-project-id
//     Severity: "severity"
// 	   LogName:
//       "mylogs": "errorlog"
//     Labels:
//       - "hostname"
//       - "service"
//
type Stackdriver struct {
	core.BufferedProducer `gollumdoc:"embed_type"`
	ProjectID             string            `config:"ProjectID"`
	LogName               map[string]string `config:"LogName"`
	Labels                []string          `config:"LabelsFrom"`
	Severity              string            `config:"Severity"`

	client *logging.Client
	logger map[core.MessageStreamID]*logging.Logger
}

func init() {
	core.TypeRegistry.Register(Stackdriver{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *Stackdriver) Configure(conf core.PluginConfigReader) {
	prod.logger = make(map[core.MessageStreamID]*logging.Logger)

	if len(prod.ProjectID) == 0 {
		conf.Errors.Pushf("A ProjectID must be provided.")
	}
}

func (prod *Stackdriver) printMessage(msg *core.Message) {
	logger, ok := prod.logger[msg.GetStreamID()]
	if !ok {
		streamName := msg.GetStreamID().GetName()
		logName, ok := prod.LogName[streamName]
		if !ok {
			logName = streamName
		}
		logger = prod.client.Logger(logName)
		prod.logger[msg.GetStreamID()] = logger
	}

	entry := logging.Entry{
		Timestamp: msg.GetCreationTime(),
		Severity:  logging.Default,
		Payload:   msg.String(),
	}

	// Parse metadata based fields
	if metadata := msg.TryGetMetadata(); metadata != nil {
		if len(prod.Severity) > 0 {
			if value, ok := metadata.Value(prod.Severity); ok {
				entry.Severity = logging.ParseSeverity(core.ConvertToString(value))
			} else {
				prod.Logger.Debugf("metadata field %s for severity not set", prod.Severity)
			}
		}

		if len(prod.Labels) > 0 {
			entry.Labels = make(map[string]string)

			for _, key := range prod.Labels {
				if value, ok := metadata.Value(key); ok {
					entry.Labels[key] = core.ConvertToString(value)
				} else {
					entry.Labels[key] = ""
					prod.Logger.Debugf("metadata field %s not set", key)
				}
			}
		}
	}

	logger.Log(entry)
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
