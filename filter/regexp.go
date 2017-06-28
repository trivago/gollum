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

package filter

import (
	"github.com/trivago/gollum/core"
	"regexp"
)

// RegExp filter plugin
//
// This plugin filters messages using regular expressions.
//
// Configuration example
//
// # Generate junk
// JunkGenerator:
//   Type: "consumer.Profiler"
//   Message: "%20s"
//   Streams: "junkstream"
//   Characters: "abcdefghijklmZ"
//   KeepRunning: true
//   Runs: 10000
//   Batches: 3000000
//   DelayMs: 500
// # Produce junk
// JunkProd00:
//   Type: "producer.Console"
//   Streams: "junkstream"
//   Modulators:
//     - "filter.RegExp":
//         Expression: "Z"
//     - "format.Envelope":
//         Prefix: "[junk_00] "
//
// FilterExpression defines the regular expression used for matching the message
// payload. If the expression matches, the message is passed.
// FilterExpression is evaluated after FilterExpressionNot.
//
// FilterExpressionNot defines a negated regular expression used for matching
// the message payload. If the expression matches, the message is blocked.
// FilterExpressionNot is evaluated before FilterExpression.
type RegExp struct {
	core.SimpleFilter `gollumdoc:"embed_type"`
	exp               *regexp.Regexp
	expNot            *regexp.Regexp
	getAppliedContent core.GetAppliedContent
}

func init() {
	core.TypeRegistry.Register(RegExp{})
}

// Configure initializes this filter with values from a plugin config.
func (filter *RegExp) Configure(conf core.PluginConfigReader) {
	var err error
	exp := conf.GetString("Expression", "")
	if exp != "" {
		filter.exp, err = regexp.Compile(exp)
		conf.Errors.Push(err)
	}

	notExp := conf.GetString("ExpressionNot", "")
	if notExp != "" {
		filter.expNot, err = regexp.Compile(notExp)
		conf.Errors.Push(err)
	}

	filter.getAppliedContent = core.GetAppliedContentGetFunction(conf.GetString("ApplyTo", ""))
}

// ApplyFilter check if all Filter wants to reject the message
func (filter *RegExp) ApplyFilter(msg *core.Message) (core.FilterResult, error) {
	if filter.expNot != nil && filter.expNot.MatchString(string(filter.getAppliedContent(msg))) {
		return filter.GetFilterResultMessageReject(), nil
	}

	if filter.exp != nil && !filter.exp.MatchString(string(filter.getAppliedContent(msg))) {
		return filter.GetFilterResultMessageReject(), nil
	}

	return core.FilterResultMessageAccept, nil
}
