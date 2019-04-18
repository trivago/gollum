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

package format

import (
	"strings"

	"github.com/mssola/user_agent"
	"github.com/trivago/gollum/core"
)

// Agent formatter
//
// This formatter parses a user agent string and outputs it as metadata fields
// to the set target.
//
// Parameters
//
// - Fields: An array of the fields to extract from the user agent.
// Available fields are: "mozilla", "platform", "os", "localization", "engine",
// "engine-version", "browser", "browser-version", "bot", "mobile".
// By default this is set to ["platform","os","localization","browser"].
//
// - Prefix: Defines a prefix for each of the keys generated.
// By default this is set to "".
//
// Examples
//
//  exampleConsumer:
//    Type: consumer.Console
//    Streams: stdin
//    Modulators:
//      - format.Agent
//        Source: user_agent
type Agent struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	prefix               string `config:"Prefix"`
	fields               []string
}

func init() {
	core.TypeRegistry.Register(Agent{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Agent) Configure(conf core.PluginConfigReader) {
	format.fields = conf.GetStringArray("Fields", []string{"platform", "os", "localization", "browser"})

	for i, field := range format.fields {
		format.fields[i] = strings.ToLower(field)
	}
}

// ApplyFormatter update message payload
func (format *Agent) ApplyFormatter(msg *core.Message) error {
	agent := user_agent.New(format.GetSourceDataAsString(msg))
	metadata := format.ForceTargetAsMetadata(msg)

	for _, field := range format.fields {
		key := format.prefix + field
		switch field {
		case "mozilla":
			metadata.Set(key, agent.Mozilla())
		case "platform":
			metadata.Set(key, agent.Platform())
		case "os":
			metadata.Set(key, agent.OS())
		case "localization":
			metadata.Set(key, agent.Localization())
		case "engine":
			name, _ := agent.Engine()
			metadata.Set(key, name)
		case "engine-version":
			_, version := agent.Engine()
			metadata.Set(key, version)
		case "browser":
			name, _ := agent.Browser()
			metadata.Set(key, name)
		case "browser-version":
			_, version := agent.Browser()
			metadata.Set(key, version)
		case "bot":
			metadata.Set(key, agent.Bot())
		case "mobile":
			metadata.Set(key, agent.Mobile())
		}
	}
	return nil
}
