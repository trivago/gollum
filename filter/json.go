// Copyright 2015-2016 trivago GmbH
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
	"github.com/trivago/gollum/shared"
	"regexp"
	"strconv"
)

// JSON allows filtering of JSON messages by looking at certain fields.
// Note that this filter is quite expensive due to JSON marshaling and regexp
// testing of every message passing through it.
// Configuration example
//
//  - "stream.Broadcast":
//    Filter: "filter.JSON"
//    FilterReject:
//      "command" : "state\d\..*"
//    FilterAccept:
//	    "args/results[0]value" : "true"
//      "args/results[1]" : "true"
//      "command" : "state\d\..*"
//
// FormatReject defines fields that will cause a message to be rejected if the
// given regular expression matches. Rejects are checked before Accepts.
// Field paths can be defined in a format accepted by shared.MarshalMap.Path.
//
// FormatAccept defines fields that will cause a message to be rejected if the
// given regular expression does not match.
// Field paths can be defined in a format accepted by shared.MarshalMap.Path.
type JSON struct {
	rejectValues map[string]*regexp.Regexp
	acceptValues map[string]*regexp.Regexp
}

func init() {
	shared.TypeRegistry.Register(JSON{})
}

// Configure initializes this filter with values from a plugin config.
func (filter *JSON) Configure(conf core.PluginConfig) error {
	rejectValues := conf.GetStringMap("FilterReject", make(map[string]string))
	acceptValues := conf.GetStringMap("FilterAccept", make(map[string]string))

	// Compile regexp from map[string]string to map[string]*regexp.Regexp
	filter.rejectValues = make(map[string]*regexp.Regexp)
	filter.acceptValues = make(map[string]*regexp.Regexp)

	for key, val := range rejectValues {
		exp, err := regexp.Compile(val)
		if err != nil {
			return err
		}
		filter.rejectValues[key] = exp
	}

	for key, val := range acceptValues {
		exp, err := regexp.Compile(val)
		if err != nil {
			return err
		}
		filter.acceptValues[key] = exp
	}

	return nil
}

func (filter *JSON) getValue(key string, values shared.MarshalMap) (string, bool) {
	if value, found := values.Path(key); found {
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

// Accepts checks JSON field values and rejects messages after testing a
// blacklist and a whitelist.
func (filter *JSON) Accepts(msg core.Message) bool {
	values := shared.NewMarshalMap()
	if err := json.Unmarshal(msg.Data, &values); err != nil {
		return false
	}

	// Check rejects
	for key, exp := range filter.rejectValues {
		if value, exists := filter.getValue(key, values); exists {
			if exp.MatchString(value) {
				return false
			}
		}
	}

	// Check accepts
	for key, exp := range filter.acceptValues {
		if value, exists := filter.getValue(key, values); exists {
			if !exp.MatchString(value) {
				return false
			}
		}
	}

	return true
}
