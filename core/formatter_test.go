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
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/trivago/tgo/ttesting"
)

type dummyFormatter struct {
	SimpleFormatter
	ConfigureHasCalled      bool
	ApplyFormatterHasCalled bool
}

func (format *dummyFormatter) Configure(conf PluginConfigReader) {
	format.ConfigureHasCalled = true
}

func (format *dummyFormatter) GetLogger() logrus.FieldLogger {
	return logrus.WithField("Scope", "dummyFormatter")
}

func (format *dummyFormatter) ApplyFormatter(msg *Message) error {
	format.ApplyFormatterHasCalled = true
	return nil
}

type dummyErrorFormatter struct {
	SimpleFormatter
	ConfigureHasCalled      bool
	ApplyFormatterHasCalled bool
}

func (format *dummyErrorFormatter) Configure(conf PluginConfigReader) {
	format.ConfigureHasCalled = true
}

func (format *dummyErrorFormatter) GetLogger() logrus.FieldLogger {
	return logrus.WithField("Scope", "dummyErrorFormatter")
}

func (format *dummyErrorFormatter) ApplyFormatter(msg *Message) error {
	format.ApplyFormatterHasCalled = true
	return errors.New("Dummy error")
}

func TestFormatterModulatorApplyFormatter(t *testing.T) {
	expect := ttesting.NewExpect(t)

	formatter, err := getDummyFormatter()
	expect.NoError(err)

	formatterModulator := NewFormatterModulator(formatter)

	msg := NewMessage(nil, []byte("test"), nil, InvalidStreamID)

	expect.NoError(formatterModulator.ApplyFormatter(msg))
	expect.True(formatter.ConfigureHasCalled)
	expect.True(formatter.ApplyFormatterHasCalled)
}

func TestFormatterModulatorApplyFormatterSkipIfEmpty(t *testing.T) {
	expect := ttesting.NewExpect(t)

	formatter, err := getDummyFormatter()
	formatter.SkipIfEmpty = true
	expect.NoError(err)

	formatterModulator := NewFormatterModulator(formatter)

	msg := NewMessage(nil, []byte(""), nil, InvalidStreamID)

	expect.NoError(formatterModulator.ApplyFormatter(msg))
	expect.True(formatter.ConfigureHasCalled)
	expect.False(formatter.ApplyFormatterHasCalled)
}

func TestFormatterModulatorModulate(t *testing.T) {
	expect := ttesting.NewExpect(t)

	formatter, err := getDummyFormatter()
	expect.NoError(err)

	formatterModulator := NewFormatterModulator(formatter)

	msg := NewMessage(nil, []byte("test"), nil, InvalidStreamID)

	expect.Equal(ModulateResultContinue, formatterModulator.Modulate(msg))
	expect.True(formatter.ConfigureHasCalled)
	expect.True(formatter.ApplyFormatterHasCalled)
}

func TestFormatterModulatorModulateError(t *testing.T) {
	expect := ttesting.NewExpect(t)

	formatter, err := getDummyErrorFormatter()
	expect.NoError(err)

	formatterModulator := NewFormatterModulator(formatter)

	msg := NewMessage(nil, []byte("test"), nil, InvalidStreamID)

	expect.Equal(ModulateResultDiscard, formatterModulator.Modulate(msg))
	expect.True(formatter.ConfigureHasCalled)
	expect.True(formatter.ApplyFormatterHasCalled)
}

func TestFormatterArray(t *testing.T) {
	expect := ttesting.NewExpect(t)

	formatter, _ := getDummyFormatter()
	secondFormatter, _ := getDummyFormatter()

	formatterArray := FormatterArray{formatter, secondFormatter}
	msg := NewMessage(nil, []byte("test"), nil, InvalidStreamID)

	expect.Nil(formatterArray.ApplyFormatter(msg))

	expect.True(formatter.ConfigureHasCalled)
	expect.True(formatter.ApplyFormatterHasCalled)
	expect.True(secondFormatter.ConfigureHasCalled)
	expect.True(secondFormatter.ApplyFormatterHasCalled)
}

func getDummyFormatter() (*dummyFormatter, error) {
	TypeRegistry.Register(dummyFormatter{ConfigureHasCalled: false, ApplyFormatterHasCalled: false})
	config := NewPluginConfig("", "core.dummyFormatter")

	plugin, err := NewPluginWithConfig(config)
	if err != nil {
		return nil, err
	}

	formatter, casted := plugin.(*dummyFormatter)
	if !casted {
		return nil, errors.New("Could not carst to dummyFormatter")
	}

	return formatter, nil
}

func getDummyErrorFormatter() (*dummyErrorFormatter, error) {
	TypeRegistry.Register(dummyErrorFormatter{ConfigureHasCalled: false, ApplyFormatterHasCalled: false})
	config := NewPluginConfig("", "core.dummyErrorFormatter")

	plugin, err := NewPluginWithConfig(config)
	if err != nil {
		return nil, err
	}

	formatter, casted := plugin.(*dummyErrorFormatter)
	if !casted {
		return nil, errors.New("Could not carst to dummyErrorFormatter")
	}

	return formatter, nil
}
