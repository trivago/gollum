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
	"strings"
)

// RegExp filter plugin
// This plugin allows filtering messages using regular expressions.
// Configuration example
//
//  - filter.RegExp:
//  	Expression: "\d+-.*"
//	ExpressionNot: "\d+-.*"
//	ApplyTo: "payload" # [payload, meta:key]
//
// FilterExpression defines the regular expression used for matching the message
// payload. If the expression matches, the message is passed.
// FilterExpression is evaluated after FilterExpressionNot.
//
// FilterExpressionNot defines a negated regular expression used for matching
// the message payload. If the expression matches, the message is blocked.
// FilterExpressionNot is evaluated before FilterExpression.
const (
	APPLY_TO_PAYLOAD  = uint8(1)
	APPLY_TO_METADATA = uint8(2)
)

type RegExp struct {
	core.SimpleFilter
	exp    *regexp.Regexp
	expNot *regexp.Regexp
	ApplyTo uint8
	filterValueFunc func (msg *core.Message) string
}

func init() {
	core.TypeRegistry.Register(RegExp{})
}

// Configure initializes this filter with values from a plugin config.
func (filter *RegExp) Configure(conf core.PluginConfigReader) error {
	var err error
	filter.SimpleFilter.Configure(conf)

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

	filter.setFilterByWithConf(conf)

	return conf.Errors.OrNil()
}

// ApplyFilter check if all Filter wants to reject the message
func (filter *RegExp) ApplyFilter(msg *core.Message) (core.FilterResult, error) {
	if filter.expNot != nil && filter.expNot.MatchString(filter.filterValueFunc(msg)) {
		return core.FilterResultMessageReject, nil
	}

	if filter.exp != nil && !filter.exp.MatchString(filter.filterValueFunc(msg)) {
		return core.FilterResultMessageReject, nil
	}

	return core.FilterResultMessageAccept, nil
}

//TODO: move public
func (filter *RegExp) setFilterByWithConf(conf core.PluginConfigReader) {
	parts := strings.Split(conf.GetString("ApplyTo", ""), ":")

	if parts[0] == "meta" {
		filter.ApplyTo = APPLY_TO_METADATA
		filter.filterValueFunc = func (msg *core.Message) string {
			return string(msg.MetaData().GetValue(parts[1], []byte("")))
		}

		return
	}

	filter.ApplyTo = APPLY_TO_PAYLOAD
	filter.filterValueFunc = func (msg *core.Message) string {
		return string(msg.Data())
	}
}