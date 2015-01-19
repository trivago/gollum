package shared

import (
	"io/ioutil"
	"launchpad.net/goyaml"
)

// Sub config for specific plugins
type PluginConfig struct {
	Enable   bool
	Settings map[string]interface{}
}

// Main config containing all plugin clonfigs
type Config struct {
	Settings map[string][]PluginConfig
}

// YAMLReader interface implementation to storing values to the internal
// configuration format
func (conf Config) SetYAML(tagType string, values interface{}) bool {
	pluginMap := values.(map[interface{}]interface{})

	// The first map contains the class name of plugin instances
	// A plugin can be configured multiple times, where each configuration
	// equals a new instance

	for pluginClass, pluginSettingInstances := range pluginMap {
		instances := pluginSettingInstances.([]interface{})
		var pluginInstances []PluginConfig

		for _, instance := range instances {
			pluginSetting := instance.(map[interface{}]interface{})
			plugin := PluginConfig{false, make(map[string]interface{})}

			// Parse the global settings from the map and set them directly at
			// the PluginConfig member. All other settings go to the Settings
			// map.

			for settingKey, settingValue := range pluginSetting {
				key := settingKey.(string)

				switch key {
				case "Enable":
					plugin.Enable = settingValue.(bool)
				default:
					plugin.Settings[key] = settingValue
				}
			}

			pluginInstances = append(pluginInstances, plugin)
		}

		conf.Settings[pluginClass.(string)] = pluginInstances
	}

	return true
}

// Read the config file into a new config struct.
func ReadConfig(path string) (*Config, error) {
	buffer, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	conf := &Config{make(map[string][]PluginConfig)}
	err = goyaml.Unmarshal(buffer, conf)

	return conf, err
}
