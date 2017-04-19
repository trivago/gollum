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

package router

import (
	"github.com/trivago/gollum/core"
	_ "github.com/trivago/gollum/filter"
	_ "github.com/trivago/gollum/format"
	"testing"
)

func TestStreamInterface(t *testing.T) {
	router := core.TypeRegistry.GetRegistered("router.")

	if len(router) == 0 {
		t.Error("No streams defined")
	}

	for _, name := range router {
		conf := core.NewPluginConfig("", name)
		_, err := core.NewPlugin(conf)
		if err != nil {
			t.Errorf("Failed to create stream %s: %s", name, err.Error())
		}
	}
}
