package shared

import "sync"

// PluginRunState is used in some plugins to store information about the
// execution state of the plugin (i.e. if it is running or not) as well as
// threading primitives that enable gollum to wait for a plugin top properly
// shut down.
type PluginRunState struct {
	Active    bool
	WaitGroup *sync.WaitGroup
}

// Plugin is the base class for any runtime class that can be configured and
// instantiated during runtim.
type Plugin interface {
	Configure(conf PluginConfig) error
}
