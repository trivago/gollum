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

package producer

import (
	"github.com/trivago/gollum/core"
	_ "github.com/trivago/gollum/filter"
	_ "github.com/trivago/gollum/format"
	"github.com/trivago/gollum/shared"
	"reflect"
	"testing"
)

func TestProducerInterface(t *testing.T) {
	conf := core.NewPluginConfig(reflect.TypeOf(t).Name())

	conf.Stream = []string{"test"}
	producers := shared.TypeRegistry.GetRegistered("producer.")

	if len(producers) == 0 {
		t.Error("No producers defined")
	}

	for _, name := range producers {
		_, err := core.NewPluginWithType(name, conf)
		if err != nil {
			t.Errorf("Failed to create producer %s: %s", name, err.Error())
		}
	}
}
