// Copyright 2015-2017 trivago GmbH
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
	"encoding/json"
	"fmt"
	"github.com/trivago/gollum/core"
	"github.com/vjeantet/grok"
)

// GrokToJSON formatter plugin
// GrokToJSON is a formatter that applies regex filters on messages.
// It works by combining text patterns into something that matches your logs.
// Configuration example
//
// - format.GrokToJSON:
//     Patterns:
//       - ^(?P<datacenter>[^\.]+?)\.(?P<service>[^\.]+?)\.(?P<host>[^\.]+?)\.statsd\.gauge-(?P<application>[^\.]+?)\.(?P<measurement>[^\s]+?)\s%{NUMBER:value_gauge:float}\s*%{INT:time}
//       - ^(?P<datacenter>[^\.]+?)\.(?P<service>[^\.]+?)\.(?P<host>[^\.]+?)\.statsd\.latency-(?P<application>[^\.]+?)\.(?P<measurement>[^\s]+?)\s%{NUMBER:value_latency:float}\s*%{INT:time}
//       - ^(?P<datacenter>[^\.]+?)\.(?P<service>[^\.]+?)\.(?P<host>[^\.]+?)\.statsd\.derive-(?P<application>[^\.]+?)\.(?P<measurement>[^\s]+?)\s%{NUMBER:value_derive:float}\s*%{INT:time}
//       - ^(?P<datacenter>[^\.]+?)\.(?P<service>[^\.]+?)\.(?P<host>[^\.]+?)\.(?P<measurement>[^\s]+?)\s%{NUMBER:value:float}\s*%{INT:time}
//
// The output format is JSON.
// For example, if using the example configuration from above and the following line as input
// `us-west.servicename.webserver0.this.is.the.measurement 12.0 1497003802`
//
// the output will be as follows:
// {
//   "datacenter": "us-west",
//   "service": "servicename",
//   "host": "webserver0",
//   "measurement": "this.is.the.measurement",
//   "value": "12.0",
//   "time": "1497003802"
// }
// Patterns is a list of grok patterns that will be executed on the given message.
// The first matching pattern will be used to parse the message.
type GrokToJSON struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	grok                 *grok.Grok
	patterns             []string `config:"Patterns"`
}

func init() {
	core.TypeRegistry.Register(GrokToJSON{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *GrokToJSON) Configure(conf core.PluginConfigReader) {
	grok, err := grok.NewWithConfig(&grok.Config{RemoveEmptyValues: true})
	if err != nil {
		conf.Errors.Push(err)
	}

	format.grok = grok
}

// ApplyFormatter update message payload
func (format *GrokToJSON) ApplyFormatter(msg *core.Message) error {
	content := format.GetAppliedContent(msg)

	values, err := format.applyGrok(string(content[:]))
	if err != nil {
		return err
	}

	serialized, err := json.Marshal(values)
	if err != nil {
		return err
	}

	format.SetAppliedContent(msg, serialized)
	return nil
}

// grok iterates over all defined patterns and parses the content based on the first match.
// It returns a map of the defined values.
func (format *GrokToJSON) applyGrok(content string) (map[string]string, error) {
	for _, pattern := range format.patterns {
		values, err := format.grok.Parse(pattern, content)

		if err == nil && len(values) > 0 {
			return values, nil
		}
	}
	format.Logger.Warningf("Message does not match any pattern: %s", content)
	return nil, fmt.Errorf("Grok parsing error")
}
