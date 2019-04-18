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
	"strconv"
	"testing"
	"time"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tcontainer"
	"github.com/trivago/tgo/ttesting"
)

func TestConvertTimeFromUnix(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.ConvertTime")
	config.Override("Source", "time")
	config.Override("ToFormat", time.RFC3339)

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*ConvertTime)
	expect.True(casted)

	timestamp := time.Now()

	msg := core.NewMessage(nil, []byte("not applied"), tcontainer.MarshalMap{"time": timestamp.Unix()}, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal(timestamp.Format(time.RFC3339), msg.String())
}

func TestConvertTimeToUnix(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.ConvertTime")
	config.Override("Source", "time")
	config.Override("FromFormat", time.RubyDate)

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*ConvertTime)
	expect.True(casted)

	timestamp := time.Now()

	msg := core.NewMessage(nil, []byte("not applied"), tcontainer.MarshalMap{"time": timestamp.Format(time.RubyDate)}, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	v, err := strconv.ParseInt(msg.String(), 10, 64)
	expect.NoError(err)

	expect.Equal(timestamp.Unix(), v)
}

func TestConvertTimeFormat(t *testing.T) {
	expect := ttesting.NewExpect(t)

	config := core.NewPluginConfig("", "format.ConvertTime")
	config.Override("Source", "time")
	config.Override("FromFormat", time.RubyDate)
	config.Override("ToFormat", time.RFC3339)

	plugin, err := core.NewPluginWithConfig(config)
	expect.NoError(err)

	formatter, casted := plugin.(*ConvertTime)
	expect.True(casted)

	timestamp := time.Now()

	msg := core.NewMessage(nil, []byte("not applied"), tcontainer.MarshalMap{"time": timestamp.Format(time.RubyDate)}, core.InvalidStreamID)

	err = formatter.ApplyFormatter(msg)
	expect.NoError(err)

	expect.Equal(timestamp.Format(time.RFC3339), msg.String())
}
