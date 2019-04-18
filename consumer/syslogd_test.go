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

package consumer

import (
	"testing"

	"github.com/trivago/tgo/tcontainer"
	"github.com/trivago/tgo/ttesting"
)

func TestSyslogStructuredDataParser(t *testing.T) {
	expect := ttesting.NewExpect(t)

	data1 := "[id@12345 key=\"value\" key2=\"quoted \\\"data\\\"\" key3=\"line\nbreak\" key4 = \"value\"]"
	metadata := tcontainer.MarshalMap{}

	parseCustomFields(data1, &metadata)
	expect.MapEqual(metadata, "key", "value")
	expect.MapEqual(metadata, "key2", "quoted \"data\"")
	expect.MapEqual(metadata, "key3", "line\nbreak")
	expect.MapEqual(metadata, "key4", "value")

	data2 := "[id@12345 key=\"value\"][id@12345 key2=\"value\"]"
	metadata = tcontainer.MarshalMap{}
	parseCustomFields(data2, &metadata)
	expect.MapEqual(metadata, "key", "value")
	expect.MapEqual(metadata, "key2", "value")
}
