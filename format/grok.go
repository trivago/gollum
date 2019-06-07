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
	"fmt"

	"github.com/trivago/gollum/core"
	"github.com/trivago/grok"
	"github.com/trivago/tgo/tcontainer"
)

// Grok formatter plugin
//
// Grok is a formatter that applies regex filters to messages and stores the result as
// metadata fields. If the target key is not existing it will be created. If the target
// key is existing but not a map, it will be replaced.
// It works by combining text patterns into something that matches your logs.
// See https://www.elastic.co/guide/en/logstash/current/plugins-filters-grok.html#_grok_basics
// for more information about Grok.
//
// Parameters
//
// - RemoveEmptyValues: When set to true, empty captures will not be returned.
// By default this parameter is set to "true".
//
// - NamedCapturesOnly: When set to true, only named captures will be returned.
// By default this parameter is set to "true".
//
// - SkipDefaultPatterns: When set to true, standard grok patterns will not be
// included in the list of patterns.
// By default this parameter is set to "true".
//
// - Patterns: A list of grok patterns that will be applied to messages.
// The first matching pattern will be used to parse the message.
//
// Examples
//
// This example transforms unstructured input into a structured json output.
// Input:
//
//  us-west.servicename.webserver0.this.is.the.measurement 12.0 1497003802
//
// Output:
//
//  {
//    "datacenter": "us-west",
//    "service": "servicename",
//    "host": "webserver0",
//    "measurement": "this.is.the.measurement",
//    "value": "12.0",
//    "time": "1497003802"
//  }
//
// Config:
//
//  exampleConsumer:
//    Type: consumer.Console
//    Streams: "*"
//    Modulators:
//      - format.Grok:
//        Patterns:
//          - ^(?P<datacenter>[^\.]+?)\.(?P<service>[^\.]+?)\.(?P<host>[^\.]+?)\.statsd\.gauge-(?P<application>[^\.]+?)\.(?P<measurement>[^\s]+?)\s%{NUMBER:value_gauge:float}\s*%{INT:time}
//          - ^(?P<datacenter>[^\.]+?)\.(?P<service>[^\.]+?)\.(?P<host>[^\.]+?)\.statsd\.latency-(?P<application>[^\.]+?)\.(?P<measurement>[^\s]+?)\s%{NUMBER:value_latency:float}\s*%{INT:time}
//          - ^(?P<datacenter>[^\.]+?)\.(?P<service>[^\.]+?)\.(?P<host>[^\.]+?)\.statsd\.derive-(?P<application>[^\.]+?)\.(?P<measurement>[^\s]+?)\s%{NUMBER:value_derive:float}\s*%{INT:time}
//          - ^(?P<datacenter>[^\.]+?)\.(?P<service>[^\.]+?)\.(?P<host>[^\.]+?)\.(?P<measurement>[^\s]+?)\s%{NUMBER:value:float}\s*%{INT:time}
//		- format.ToJSON: {}
type Grok struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	exp                  []*grok.CompiledGrok
}

func init() {
	core.TypeRegistry.Register(Grok{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *Grok) Configure(conf core.PluginConfigReader) {
	grokParser, err := grok.New(grok.Config{
		RemoveEmptyValues:   conf.GetBool("RemoveEmptyValues", true),
		NamedCapturesOnly:   conf.GetBool("NamedCapturesOnly", true),
		SkipDefaultPatterns: conf.GetBool("SkipDefaultPatterns", false),
	})
	if err != nil {
		conf.Errors.Push(err)
	}

	//format.grok = grok
	patterns := conf.GetStringArray("Patterns", []string{})
	for _, p := range patterns {
		exp, err := grokParser.Compile(p)
		if err != nil {
			conf.Errors.Push(err)
		}
		format.exp = append(format.exp, exp)
	}
}

// ApplyFormatter update message payload
func (format *Grok) ApplyFormatter(msg *core.Message) error {
	metadata := format.ForceTargetAsMetadata(msg)
	content := format.GetSourceDataAsString(msg)

	return format.applyGrok(metadata, content)
}

// grok iterates over all defined patterns and parses the content based on the first match.
// It returns a map of the defined values.
func (format *Grok) applyGrok(metadata tcontainer.MarshalMap, content string) error {
	for _, exp := range format.exp {
		values := exp.ParseString(content)

		if len(values) > 0 {
			for k, v := range values {
				metadata.Set(k, v)
			}
			return nil
		}
	}
	format.Logger.Warningf("Message does not match any pattern: %s", content)
	return fmt.Errorf("grok parsing error")
}
