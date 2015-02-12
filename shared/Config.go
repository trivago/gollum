package shared

import (
	"io/ioutil"
	"launchpad.net/goyaml"
	"reflect"
)

// PluginConfig is a configuration for a specific plugin
type PluginConfig struct {
	Enable   bool
	Channel  int
	Stream   []string
	Settings map[string]interface{}
}

// Config represents the top level config containing all plugin clonfigs
type Config struct {
	Settings map[string][]PluginConfig
}

// SetYAML is a YAMLReader interface implementation to convert values into the
// internal configuration format
func (conf Config) SetYAML(tagType string, values interface{}) bool {
	pluginList := values.([]interface{})
	stringType := reflect.TypeOf("")

	// As there might be multiple instances of the same plugin class we iterate
	// over an array here.

	for _, pluginData := range pluginList {
		pluginDataMap := pluginData.(map[interface{}]interface{})

		// Each item in the array item is a map{class -> map{key -> value}}
		// We "iterate" over the first map (one item only) to get the class.

		for pluginClass, pluginSettings := range pluginDataMap {
			pluginSettingsMap := pluginSettings.(map[interface{}]interface{})

			plugin := PluginConfig{
				Enable:   true,
				Channel:  4096,
				Stream:   []string{},
				Settings: make(map[string]interface{}),
			}

			// Iterate over all key/value pairs.
			// "Enable" is a special field as non-plugin logic is bound to it

			for settingKey, settingValue := range pluginSettingsMap {
				key := settingKey.(string)

				switch key {
				case "Enable":
					plugin.Enable = settingValue.(bool)

				case "Channel":
					plugin.Channel = settingValue.(int)

				case "Stream":
					if reflect.TypeOf(settingValue) == stringType {
						plugin.Stream = append(plugin.Stream, settingValue.(string))
					} else {
						for _, value := range settingValue.([]interface{}) {
							plugin.Stream = append(plugin.Stream, value.(string))
						}
					}

				default:
					plugin.Settings[key] = settingValue
				}
			}

			// Set wildcard stream if no stream is set

			if len(plugin.Stream) == 0 {
				plugin.Stream = append(plugin.Stream, "*")
			}

			// Add instance of this plugin class config to the list

			list, listExists := conf.Settings[pluginClass.(string)]
			if !listExists {
				list = []PluginConfig{}
			}

			conf.Settings[pluginClass.(string)] = append(list, plugin)
		}
	}

	return true
}

// HasValue returns true if the given key has been set as a config option.
// This function only takes non-predefined settings into account.
func (conf PluginConfig) HasValue(key string) bool {
	_, exists := conf.Settings[key]
	return exists
}

// Override sets or override a configuration value for non-predefined options.
func (conf PluginConfig) Override(key string, value interface{}) {
	conf.Settings[key] = value
}

// GetString tries to read a non-predefined, string value from a PluginConfig.
// If that value is not found defaultValue is returned.
func (conf PluginConfig) GetString(key string, defaultValue string) string {
	value, exists := conf.Settings[key]
	if exists {
		return value.(string)
	}

	return defaultValue
}

// GetStringArray tries to read a non-predefined, string array from a
// PluginConfig. If that value is not found defaultValue is returned.
func (conf PluginConfig) GetStringArray(key string, defaultValue []string) []string {
	value, exists := conf.Settings[key]
	if exists {
		stringValue, isSingleValue := value.(string)
		if isSingleValue {
			return []string{stringValue}
		}

		arrayValue := value.([]interface{})
		config := make([]string, 0, len(arrayValue))
		for _, value := range arrayValue {
			config = append(config, value.(string))
		}
		return config
	}

	return defaultValue
}

// GetStringMap tries to read a non-predefined, string to string map from a
// PluginConfig. If that value is not found defaultValue is returned.
func (conf PluginConfig) GetStringMap(key string, defaultValue map[string]string) map[string]string {
	mapping, exists := conf.Settings[key]
	if !exists {
		return defaultValue
	}

	result := make(map[string]string)
	for key, value := range mapping.(map[interface{}]interface{}) {
		result[key.(string)] = value.(string)
	}

	return result
}

// GetStreamMap tries to read a non-predefined, stream to string map from a
// plugin config. A mapping on the wildcard stream is always returned.
// The target is either defaultValue or a value defined by the config.
func (conf PluginConfig) GetStreamMap(key string, defaultValue string) map[MessageStreamID]string {
	streamMap := make(map[MessageStreamID]string)
	streamMap[WildcardStreamID] = defaultValue

	value, exists := conf.Settings[key]
	if !exists {
		return streamMap
	}

	for streamName, target := range value.(map[interface{}]interface{}) {
		streamMap[GetStreamID(streamName.(string))] = target.(string)
	}

	return streamMap
}

// GetInt tries to read a non-predefined, integer value from a PluginConfig.
// If that value is not found defaultValue is returned.
func (conf PluginConfig) GetInt(key string, defaultValue int) int {
	value, exists := conf.Settings[key]
	if exists {
		return value.(int)
	}

	return defaultValue
}

// GetBool tries to read a non-predefined, boolean value from a PluginConfig.
// If that value is not found defaultValue is returned.
func (conf PluginConfig) GetBool(key string, defaultValue bool) bool {
	value, exists := conf.Settings[key]
	if exists {
		return value.(bool)
	}

	return defaultValue
}

// GetValue tries to read a non-predefined, untyped value from a PluginConfig.
// If that value is not found defaultValue is returned.
func (conf PluginConfig) GetValue(key string, defaultValue interface{}) interface{} {
	value, exists := conf.Settings[key]
	if exists {
		return value
	}

	return defaultValue
}

// ReadConfig parses a YAML config file into a new Config struct.
func ReadConfig(path string) (*Config, error) {
	buffer, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	conf := &Config{make(map[string][]PluginConfig)}
	err = goyaml.Unmarshal(buffer, conf)

	return conf, err
}
