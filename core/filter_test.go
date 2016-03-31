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

package core

import (
	"github.com/trivago/tgo/tlog"
)

type mockFilter struct {
}

func (mf *mockFilter) SetLogScope(log tlog.LogScope) {
}

func (mf *mockFilter) Accepts(msg Message) bool {
	return true
}

func (mf *mockFilter) Drop(msg Message) {
}

func (mf *mockFilter) Configure(conf PluginConfig) error {
	return nil
}
