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

// JSON filter
//
// This filter allows inspecting fields of JSON encoded datasets and accepting
// or rejecting messages based on their contents.
//
// Parameters
//
// - Reject: Defines fields that will cause a message to be rejected if the
// given regular expression matches. Reject is checked before Accept.
// Field paths can be defined in a format accepted by tgo.MarshalMap.Path.
// By default this parameter is set to an empty list.
//
// - Accept: Defines fields that will cause a message to be rejected if the
// given regular expression does not match. Accept is checked after Reject.
// Field paths can be defined in a format accepted by tgo.MarshalMap.Path.
// By default this parameter is set to an empty list.
//
// - ApplyTo: Defines which part of the message is affected by the filter.
// When setting this parameter to "" this filter is applied to the
// message payload. Every other value denotes a metadata key.
// By default this parameter is set to "".
//
// Examples
//
//  ExampleConsumer:
//    Type: consumer.Console
//    Streams: console
//    Modulators:
//      - filter.JSON:
//          Reject:
//            type: ^log\.
//          Accept:
//            source: ^www\d+\.
//            data/active: true
//
type JSON struct {
	core.SimpleFilter `gollumdoc:"embed_type"`
	rejectValues      map[string]*regexp.Regexp
	acceptValues      map[string]*regexp.Regexp
	getAppliedContent core.GetAppliedContent
}

func init() {
	core.TypeRegistry.Register(JSON{})
}

// Configure initializes this filter with values from a plugin config.
func (filter *JSON) Configure(conf core.PluginConfigReader) {
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

	applyTo := conf.GetString("ApplyTo", "")
	filter.getAppliedContent = core.GetAppliedContentGetFunction(applyTo)
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
