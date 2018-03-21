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

package filter

import (
	"github.com/trivago/gollum/core"
	"regexp"
)

// RegExp filter
//
// This filter rejects or accepts messages based on regular expressions.
//
// Parameters
//
// - FilterExpression: Messages matching this expression are passed on.
// This parameter is ignored when set to "". FilterExpression is checked
// after FilterExpressionNot.
// By default this parameter is set to "".
//
// - FilterExpressionNot: Messages *not* matching this expression are
// passed on. This parameter is ignored when set to "". FilterExpressionNot
// is checked before FilterExpression.
// By default this parameter is set to "".
//
// - ApplyTo: Defines which part of the message the filter is applied to.
// When set to "", this filter is applied to the message's payload. All
// other values denotes a metadata key.
// By default this parameter is set to "".
//
// Examples
//
// This example accepts only accesslog entries with a return status of
// 2xx or 3xx not originated from staging systems.
//
//  ExampleConsumer:
//    Type: consumer.Console
//    Streams: console
//    Modulators:
//      - filter.RegExp:
//        FilterExpressionNot: " stage\\."
//        FilterExpression: "HTTP/1\\.1\\\" [23]\\d\\d"
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
