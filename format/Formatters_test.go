// Copyright 2015 trivago GmbH
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
	"github.com/trivago/gollum/shared"
	"testing"
)

func testFormatter(t *testing.T, formatter shared.Formatter) bool {
	expect := shared.NewExpect(t)

	msgString := "formatter" //\ttest\r\n123 456\n"
	msg := shared.NewMessage(msgString, []shared.MessageStreamID{}, 0)

	formatter.PrepareMessage(msg)
	buffer := make([]byte, formatter.GetLength())
	result := true

	result = expect.IntEq(formatter.GetLength(), len(formatter.String())) && result
	result = expect.IntEq(formatter.GetLength(), formatter.CopyTo(buffer)) && result
	result = expect.StringEq(formatter.String(), string(buffer)) && result

	return result
}

func TestFormatters(t *testing.T) {
	conf := shared.PluginConfig{}
	formatters := shared.RuntimeType.GetRegistered("format.")

	if len(formatters) == 0 {
		t.Error("No formatters defined")
	}

	for _, name := range formatters {
		plugin, err := shared.RuntimeType.NewPlugin(name, conf)
		if err != nil {
			t.Errorf("Failed to create formatter %s", name)
		} else {
			if !testFormatter(t, plugin.(shared.Formatter)) {
				t.Errorf("Formatter %s tests failed", name)
			}
		}
	}
}
