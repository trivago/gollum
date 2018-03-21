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
	"fmt"
	"io/ioutil"
	"reflect"

	"github.com/sirupsen/logrus"
	"github.com/trivago/tgo"
	"github.com/trivago/tgo/tcontainer"
	"github.com/trivago/tgo/treflect"
	"gopkg.in/yaml.v2"
)

const pluginAggregate = "Aggregate"

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

// ReadConfig creates a config from a yaml byte stream.
func ReadConfig(buffer []byte) (*Config, error) {
	config := new(Config)
	if err := yaml.Unmarshal(buffer, &config.Values); err != nil {
		return nil, err
	}

	// As there might be multiple instances of the same plugin class we iterate
	// over an array here.
	hasError := false
	for pluginID, configValues := range config.Values {
		if typeName, _ := configValues.String("Type"); typeName == pluginAggregate {
			// aggregate behavior
			aggregateMap, err := configValues.MarshalMap("Plugins")
			if err != nil {
				hasError = true
				logrus.Error("Can't read 'Aggregate' configuration: ", err)
				continue
			}

			// loop through aggregated plugins and set them up
			for subPluginID, subConfigValues := range aggregateMap {
				subPluginsID := fmt.Sprintf("%s-%s", pluginID, subPluginID)
				subConfig, err := tcontainer.ConvertToMarshalMap(subConfigValues, nil)
				if err != nil {
					hasError = true
					logrus.Error("Error in plugin config ", subPluginsID, err)
					continue
				}

				// set up sub-plugin
				delete(configValues, "Type")
				delete(configValues, "Plugins")

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

	if hasError {
		return config, fmt.Errorf("Configuration parsing produced errors")
	}
	return config, nil
}

// ReadConfigFromFile parses a YAML config file into a new Config struct.
func ReadConfigFromFile(path string) (*Config, error) {
	buffer, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return ReadConfig(buffer)
}

// Validate checks all plugin configs and plugins on validity. I.e. it checks
// on mandatory fields and correct implementation of consumer, producer or
// stream interface. It does NOT call configure for each plugin.
func (conf *Config) Validate() error {
	errors := tgo.NewErrorStack()
	errors.SetFormat(tgo.ErrorStackFormatCSV)

	for _, config := range conf.Plugins {
		if config.Typename == "" {
			errors.Pushf("Plugin type is not set for '%s'", config.ID)
			continue
		}

		pluginType := TypeRegistry.GetTypeOf(config.Typename)
		if pluginType == nil {
			if suggestion := suggestType(config.Typename); suggestion != "" {
				errors.Pushf("Type '%s' used for '%s' not found. Did you mean '%s'?", config.Typename, config.ID, suggestion)
			} else {
				errors.Pushf("Type '%s' used for '%s' not found", config.Typename, config.ID)
			}

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

		errors.Pushf("Type '%s' used for '%s' does not implement a common interface", config.Typename, config.ID)
		getClosestMatch(pluginType, &errors)
	}

	return errors.OrNil()
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
