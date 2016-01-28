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

package core

import (
	"github.com/trivago/tgo/tcontainer"
	"github.com/trivago/tgo/tlog"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

// Config represents the top level config containing all plugin clonfigs
type Config struct {
	Values  map[string]tcontainer.MarshalMap
	Plugins []PluginConfig
}

// NewLogScope creates a new tlog.LogScope for the plugin contained in this
// config.
func NewLogScope(conf *PluginConfig) tlog.LogScope {
	if conf.ID != "" {
		return tlog.NewLogScope(conf.Typename)
	}

	return tlog.NewLogScope(conf.Typename + ":" + conf.ID)
}

// ReadConfig parses a YAML config file into a new Config struct.
func ReadConfig(path string) (*Config, error) {
	buffer, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := new(Config)
	err = yaml.Unmarshal(buffer, &config.Values)
	if err != nil {
		return nil, err
	}

	// As there might be multiple instances of the same plugin class we iterate
	// over an array here.
	for pluginID, configValues := range config.Values {
		pluginConfig := NewPluginConfig(pluginID, "")
		pluginConfig.Read(configValues)
		config.Plugins = append(config.Plugins, pluginConfig)
	}

	return config, err
}
