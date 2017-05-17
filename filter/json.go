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
	"encoding/json"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tcontainer"
	"regexp"
	"strconv"
)

// JSON filter plugin
// This plugin allows filtering of JSON messages by looking at certain fields.
// Note that this filter is quite expensive due to JSON marshaling and regexp
// testing of every message passing through it.
// Configuration example
//
//  - filter.JSON:
//    	Reject:
//        "command" : "state\d\..*"
//    	Accept:
//        "args/results[0]value" : "true"
//        "args/results[1]" : "true"
//        "command" : "state\d\..*"
//    	ApplyTo: "payload" # payload or <metaKey>
//
// FilterReject defines fields that will cause a message to be rejected if the
// given regular expression matches. Rejects are checked before Accepts.
// Field paths can be defined in a format accepted by tgo.MarshalMap.Path.
//
// FilterAccept defines fields that will cause a message to be rejected if the
// given regular expression does not match.
// Field paths can be defined in a format accepted by tgo.MarshalMap.Path.
type JSON struct {
	core.SimpleFilter `gollumdoc:embed_type`
	rejectValues      map[string]*regexp.Regexp
	acceptValues      map[string]*regexp.Regexp
	getAppliedContent core.GetAppliedContent
}

func init() {
	core.TypeRegistry.Register(JSON{})
}

// Configure initializes this filter with values from a plugin config.
func (filter *JSON) Configure(conf core.PluginConfigReader) error {
	filter.SimpleFilter.Configure(conf)

	rejectValues := conf.GetStringMap("Reject", make(map[string]string))
	acceptValues := conf.GetStringMap("Accept", make(map[string]string))

	// Compile regexp from map[string]string to map[string]*regexp.Regexp
	filter.rejectValues = make(map[string]*regexp.Regexp)
	filter.acceptValues = make(map[string]*regexp.Regexp)

	for key, val := range rejectValues {
		exp, err := regexp.Compile(val)
		if !conf.Errors.Push(err) {
			filter.rejectValues[key] = exp
		}
	}

	for key, val := range acceptValues {
		exp, err := regexp.Compile(val)
		if !conf.Errors.Push(err) {
			filter.acceptValues[key] = exp
		}
	}

	filter.getAppliedContent = core.GetAppliedContentFunction(conf.GetString("ApplyTo", ""))

	return conf.Errors.OrNil()
}

func (filter *JSON) getValue(key string, values tcontainer.MarshalMap) (string, bool) {
	if value, found := values.Value(key); found {
		switch value.(type) {
		case string:
			return value.(string), true

		case bool:
			return strconv.FormatBool(value.(bool)), true

		case float64:
			return strconv.FormatFloat(value.(float64), 'f', -1, 64), true
		}
	}

	return "", false
}

// ApplyFilter check if all Filter wants to reject the message
func (filter *JSON) ApplyFilter(msg *core.Message) (core.FilterResult, error) {
	values := tcontainer.NewMarshalMap()
	if err := json.Unmarshal(filter.getAppliedContent(msg), &values); err != nil {
		return filter.GetFilterResultMessageReject(), err
	}

	// Check rejects
	for key, exp := range filter.rejectValues {
		if value, exists := filter.getValue(key, values); exists {
			if exp.MatchString(value) {
				return filter.GetFilterResultMessageReject(), nil
			}
		}
	}

	// Check accepts
	for key, exp := range filter.acceptValues {
		if value, exists := filter.getValue(key, values); exists {
			if !exp.MatchString(value) {
				return filter.GetFilterResultMessageReject(), nil
			}
		}
	}

	return core.FilterResultMessageAccept, nil
}
