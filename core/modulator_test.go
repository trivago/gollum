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

package core

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/trivago/tgo/ttesting"
)

func TestModulate(t *testing.T) {
	expect := ttesting.NewExpect(t)

	TypeRegistry.Register(mockFormatter{})

	mockConf := NewPluginConfig("", "core.mockPlugin")
	mockConf.Override("Modulators", []interface{}{
		"core.mockFormatter",
	})

	reader := NewPluginConfigReaderWithError(&mockConf)

	modulatorArray, err := reader.GetModulatorArray("Modulators", logrus.StandardLogger(), []Modulator{})
	expect.NoError(err)

	msg := NewMessage(nil, []byte("foo"), nil, InvalidStreamID)
	expect.Equal(ModulateResultContinue, modulatorArray.Modulate(msg))
}
