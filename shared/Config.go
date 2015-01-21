package shared

import (
	"io/ioutil"
	"launchpad.net/goyaml"
	"reflect"
)

// Sub config for specific plugins
type PluginConfig struct {
	Enable   bool
	Buffer   int
	Stream   []string
	Settings map[string]interface{}
}

// Main config containing all plugin clonfigs
type Config struct {
	Settings map[string][]PluginConfig
}

// YAMLReader interface implementation to storing values to the internal
// configuration format
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

			//fmt.Println(pluginClass)
			plugin := PluginConfig{
				Enable:   false,
				Buffer:   1024,
				Stream:   make([]string, 0),
				Settings: make(map[string]interface{}),
			}

			// Iterate over all key/value pairs.
			// "Enable" is a special field as non-plugin logic is bound to it

			for settingKey, settingValue := range pluginSettingsMap {
				key := settingKey.(string)

				//fmt.Printf("\t%s -> %v\n", key, settingValue)

				switch key {
				case "Enable":
					plugin.Enable = settingValue.(bool)

				case "Buffer":
					plugin.Buffer = settingValue.(int)

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
				list = make([]PluginConfig, 0)
			}

			conf.Settings[pluginClass.(string)] = append(list, plugin)
		}
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
