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

package producer

import (
	"fmt"
	"runtime/debug"
	"testing"

	"github.com/trivago/gollum/core"
	_ "github.com/trivago/gollum/filter"
	_ "github.com/trivago/gollum/format"
	_ "github.com/trivago/gollum/router"
)

func TestProducerInterface(t *testing.T) {
	producers := core.TypeRegistry.GetRegistered("producer.")
	if len(producers) == 0 {
		t.Error("No producers defined")
	}

	name := ""
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Panic while testing %s\n%v\n\n%s", name, r, string(debug.Stack()))
		}
	}()

	var idx int
	for idx, name = range producers {
		conf := core.NewPluginConfig(fmt.Sprintf("prod%d", idx), name)
		_, err := core.NewPluginWithConfig(conf)
		if err != nil {
			t.Errorf("Failed to create producer %s: %s", name, err.Error())
		}
	}
}
