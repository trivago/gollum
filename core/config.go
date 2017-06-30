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

package core

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/trivago/tgo"
	"github.com/trivago/tgo/tcontainer"
	"github.com/trivago/tgo/treflect"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"reflect"
	"strings"
)

const pluginAggregate = "aggregate"

var (
	consumerInterface = reflect.TypeOf((*Consumer)(nil)).Elem()
	producerInterface = reflect.TypeOf((*Producer)(nil)).Elem()
	routerInterface   = reflect.TypeOf((*Router)(nil)).Elem()
)

// Config represents the top level config containing all plugin clonfigs
type Config struct {
	Values  map[string]tcontainer.MarshalMap
	Plugins []PluginConfig
}

// PluginConfigError is a container for errors produced by Config.Validate
type PluginConfigError struct {
	id       string
	typename string
	reason   string
}

// ReadConfig creates a config from a yaml byte stream.
func ReadConfig(buffer []byte) (*Config, error) {
	config := new(Config)
	err := yaml.Unmarshal(buffer, &config.Values)
	if err != nil {
		return nil, err
	}

	// As there might be multiple instances of the same plugin class we iterate
	// over an array here.
	for pluginID, configValues := range config.Values {
		if typeName, _ := configValues.String("Type"); typeName == pluginAggregate {
			// aggregate behavior
			aggregateMap, err := configValues.MarshalMap("Aggregate")
			if err != nil {
				logrus.Error("Can't read 'Aggregate' configuration: ", err)
				continue
			}

			// loop through aggregated plugins and set them up
			for subPluginID, subConfigValues := range aggregateMap {
				subPluginsID := fmt.Sprintf("%s-%s", pluginID, subPluginID)
				subConfig, err := tcontainer.ConvertToMarshalMap(subConfigValues, strings.ToLower)
				if err != nil {
					logrus.Error("Error in plugin config ", subPluginsID, err)
					continue
				}

				// set up sub-plugin
				delete(configValues, "Type")
				delete(configValues, "Aggregate")

				pluginConfig := NewPluginConfig(subPluginsID, "")
				pluginConfig.Read(configValues)
				pluginConfig.Read(subConfig)

				config.Plugins = append(config.Plugins, pluginConfig)
			}
		} else {
			// default behavior
			pluginConfig := NewPluginConfig(pluginID, "")
			pluginConfig.Read(configValues)
			config.Plugins = append(config.Plugins, pluginConfig)
		}
	}

	return config, err
}

// ReadConfigFromFile parses a YAML config file into a new Config struct.
func ReadConfigFromFile(path string) (*Config, error) {
	buffer, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return ReadConfig(buffer)
}

func (err PluginConfigError) Error() string {
	if err.typename == "" {
		return fmt.Sprintf("%s: %s", err.id, err.reason)
	}
	return fmt.Sprintf("%s (%s): %s", err.id, err.typename, err.reason)
}

// Validate checks all plugin configs and plugins on validity. I.e. it checks
// on mandatory fields and correct implementation of consumer, producer or
// stream interface. It does NOT call configure for each plugin.
func (conf *Config) Validate() []error {
	errors := tgo.NewErrorStack()

	for _, config := range conf.Plugins {
		if config.Typename == "" {
			errors.Push(newPluginConfigError(config.ID, "", "Plugin type is not set."))
			continue
		}

		pluginType := TypeRegistry.GetTypeOf(config.Typename)
		if pluginType == nil {
			errors.Push(newPluginConfigError(config.ID, config.Typename, "Type not registered. Please check compiled plugins."))
			continue // ### continue ###
		}

		switch {
		case pluginType.Implements(consumerInterface):
			continue

		case pluginType.Implements(producerInterface):
			continue

		case pluginType.Implements(routerInterface):
			continue
		}

		errors.Push(newPluginConfigError(config.ID, config.Typename, "Type does not implement a common interface."))
		getClosestMatch(pluginType, &errors)
	}

	return errors.Errors()
}

// GetConsumers returns all consumer plugins from the config
func (conf *Config) GetConsumers() []PluginConfig {
	configs := []PluginConfig{}

	for _, config := range conf.Plugins {
		if !config.Enable || config.Typename == "" {
			continue // ### continue, disabled ###
		}

		pluginType := TypeRegistry.GetTypeOf(config.Typename)
		if pluginType == nil {
			continue // ### continue, unknown type ###
		}

		if pluginType.Implements(consumerInterface) {
			logrus.Debug("Found consumer ", config.ID)
			configs = append(configs, config)
		}
	}

	return configs
}

// GetProducers returns all producer plugins from the config
func (conf *Config) GetProducers() []PluginConfig {
	configs := []PluginConfig{}

	for _, config := range conf.Plugins {
		if !config.Enable || config.Typename == "" {
			continue // ### continue, disabled ###
		}

		pluginType := TypeRegistry.GetTypeOf(config.Typename)
		if pluginType == nil {
			continue // ### continue, unknown type ###
		}

		if pluginType.Implements(producerInterface) {
			logrus.Debug("Found producer ", config.ID)
			configs = append(configs, config)
		}
	}

	return configs
}

// GetRouters returns all stream plugins from the config
func (conf *Config) GetRouters() []PluginConfig {
	configs := []PluginConfig{}

	for _, config := range conf.Plugins {
		if !config.Enable || config.Typename == "" {
			continue // ### continue, disabled ###
		}

		pluginType := TypeRegistry.GetTypeOf(config.Typename)
		if pluginType == nil {
			continue // ### continue, unknown type ###
		}

		if pluginType.Implements(routerInterface) {
			logrus.Debugf("Found router '%s'", config.ID)
			configs = append(configs, config)
		}
	}

	return configs
}

func newPluginConfigError(id string, typename string, reason string) PluginConfigError {
	return PluginConfigError{
		id:       id,
		typename: typename,
		reason:   reason,
	}
}

func getClosestMatch(pluginType reflect.Type, errors *tgo.ErrorStack) {
	consumerMatch, consumerMissing := treflect.GetMissingMethods(pluginType, consumerInterface)
	producerMatch, producerMissing := treflect.GetMissingMethods(pluginType, producerInterface)
	routerMatch, streamMissing := treflect.GetMissingMethods(pluginType, routerInterface)

	switch {
	case consumerMatch > producerMatch && consumerMatch > routerMatch:
		errors.Pushf("Plugin looks like a consumer")
		for _, missing := range consumerMissing {
			errors.Pushf(missing)
		}

	case producerMatch > consumerMatch && producerMatch > routerMatch:
		errors.Pushf("Plugin looks like a producer")
		for _, missing := range producerMissing {
			errors.Pushf(missing)
		}

	default:
		errors.Pushf("Plugin looks like a stream")
		for _, missing := range streamMissing {
			errors.Pushf(missing)
		}
	}
}
