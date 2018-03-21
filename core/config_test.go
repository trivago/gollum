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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/trivago/tgo/ttesting"
)

type TypeMockA struct {
	SimpleConsumer
}

func (mock *TypeMockA) Consume(workers *sync.WaitGroup) {
}

type TypeMockB struct {
	SimpleRouter
}

func (router *TypeMockB) Start() error {
	return nil
}

func (router *TypeMockB) Enqueue(msg *Message) error {
	return nil
}

type TypeMockC struct {
	SimpleProducer
}

func (mock *TypeMockC) Produce(workers *sync.WaitGroup) {
	//do something
}

func (mock *TypeMockC) Enqueue(msg *Message, timeout time.Duration) {
	//do something
}

func TestReadConfig(t *testing.T) {
	expect := ttesting.NewExpect(t)
	testConfig := []byte("someId: {Type: consumer.Console, Streams: foo}")

	conf, err := ReadConfig(testConfig)
	expect.NoError(err)

	for pluginID, configValues := range conf.Values {
		expect.Equal("someId", pluginID)

		value, err := configValues.String("Type")
		expect.NoError(err)
		expect.Equal("consumer.Console", value)

		value, err = configValues.String("Streams")
		expect.NoError(err)
		expect.Equal("foo", value)

		break
	}
}

func TestReadConfigError(t *testing.T) {
	expect := ttesting.NewExpect(t)
	testConfig := []byte("no yaml")

	_, err := ReadConfig(testConfig)
	expect.NotNil(err)
	expect.True(strings.Contains(err.Error(), "cannot unmarshal"))
}

func TestReadConfigWithAggregation(t *testing.T) {
	expect := ttesting.NewExpect(t)
	testConfig := []byte("someId: {Type: Aggregate, Streams: foo, Plugins: {anotherId: {Type: consumer.Console}, secondId: {Type: consumer.Console}}}")

	conf, err := ReadConfig(testConfig)
	expect.NoError(err)

	expect.Equal(2, len(conf.Plugins))

	inheritStream, err := conf.Plugins[0].Settings.String("Streams")
	expect.NoError(err)
	expect.Equal("foo", inheritStream)
}

func TestValidate(t *testing.T) {
	expect := ttesting.NewExpect(t)

	testConfig := []byte("consumerId: {Type: core.TypeMockA, Streams: foo}\nrouterId: {Type: core.TypeMockB, Streams: foo}")

	TypeRegistry.Register(TypeMockA{})
	TypeRegistry.Register(TypeMockB{})

	conf, err := ReadConfig(testConfig)
	expect.NoError(err)

	err = conf.Validate()
	expect.NoError(err)
}

func TestValidateFailure(t *testing.T) {
	expect := ttesting.NewExpect(t)

	testConfig := []byte("consumerId: {Type: unknown, Stream: foo}")

	conf, err := ReadConfig(testConfig)
	expect.NoError(err)

	err = conf.Validate()
	expect.NotNil(err)
}

func TestConfigGetPluginTypeMethods(t *testing.T) {
	expect := ttesting.NewExpect(t)

	testConfig := []byte("consumerId: {Type: core.TypeMockA, Streams: foo}\nrouterId: {Type: core.TypeMockB, Stream: foo}\nproducerId: {Type: core.TypeMockC, Streams: foo}")

	TypeRegistry.Register(TypeMockA{})
	TypeRegistry.Register(TypeMockB{})
	TypeRegistry.Register(TypeMockC{})

	conf, err := ReadConfig(testConfig)
	expect.NoError(err)

	consumers := conf.GetConsumers()
	expect.Equal(1, len(consumers))

	for _, pluginConf := range consumers {
		expect.Equal("consumerId", pluginConf.ID)
		expect.Equal("core.TypeMockA", pluginConf.Typename)
	}

	producers := conf.GetProducers()
	expect.Equal(1, len(producers))

	for _, pluginConf := range producers {
		expect.Equal("producerId", pluginConf.ID)
		expect.Equal("core.TypeMockC", pluginConf.Typename)
	}

	routers := conf.GetRouters()
	expect.Equal(1, len(routers))

	for _, pluginConf := range routers {
		expect.Equal("routerId", pluginConf.ID)
		expect.Equal("core.TypeMockB", pluginConf.Typename)
	}
}
