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
	"strconv"
	"time"

	"github.com/trivago/gollum/core"
)

// ConvertTime formatter
//
// This formatter converts one time format in another.
//
// - From: When left empty, a unix time is expected. Otherwise a go compatible
// timestamp has to be given. See https://golang.org/pkg/time/#pkg-constants
// By default this is set to "".
//
// - To: When left empty, the output will be unixtime. Otherwise a go compatible
// timestamp has to be given. See https://golang.org/pkg/time/#pkg-constants
// By default this is set to "".
//
// Examples
//
// This example removes the "pipe" key from the metadata produced by
// consumer.Console.
//
//  exampleConsumer:
//    Type: consumer.Console
//    Streams: stdin
//    Modulators:
//      - format.ConvertTime:
//        FromFormat: ""
//        ToFormat: ""
//
type ConvertTime struct {
	core.SimpleFormatter `gollumdoc:"embed_type"`
	from                 string `config:"FromFormat"`
	to                   string `config:"ToFormat"`
}

func init() {
	core.TypeRegistry.Register(ConvertTime{})
}

// Configure initializes this formatter with values from a plugin config.
func (format *ConvertTime) Configure(conf core.PluginConfigReader) {
}

func numberToUnixtime(number interface{}) (time.Time, error) {
	switch v := number.(type) {
	case int:
		return time.Unix(int64(v), 0), nil
	case int8:
		return time.Unix(int64(v), 0), nil
	case uint8:
		return time.Unix(int64(v), 0), nil
	case int16:
		return time.Unix(int64(v), 0), nil
	case uint16:
		return time.Unix(int64(v), 0), nil
	case int32:
		return time.Unix(int64(v), 0), nil
	case uint32:
		return time.Unix(int64(v), 0), nil
	case uint64:
		return time.Unix(int64(v), 0), nil
	case float32:
		return time.Unix(int64(v), 0), nil
	case float64:
		return time.Unix(int64(v), 0), nil
	case int64:
		return time.Unix(v, 0), nil

	case string:
		intVal, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return time.Time{}, err
		}
		return time.Unix(intVal, 0), nil

	case []byte:
		intVal, err := strconv.ParseInt(string(v), 10, 64)
		if err != nil {
			return time.Time{}, err
		}
		return time.Unix(intVal, 0), nil

	default:
		return time.Time{}, fmt.Errorf("cannot convert %T to int64", v)
	}
}

// ApplyFormatter update message payload
func (format *ConvertTime) ApplyFormatter(msg *core.Message) error {
	var (
		t   time.Time
		err error
	)

	if format.from == "" {
		t, err = numberToUnixtime(format.GetSourceData(msg))
	} else {
		t, err = time.Parse(format.from, format.GetSourceDataAsString(msg))
	}

	if err != nil {
		return err
	}

	if format.to == "" {
		format.SetTargetData(msg, t.Unix())
	} else {
		format.SetTargetData(msg, t.Format(format.to))
	}
	return nil
}
