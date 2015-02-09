package shared

import "sync"

type PluginRunState struct {
	Active    bool
	WaitGroup *sync.WaitGroup
}

type Plugin interface {
	Configure(conf PluginConfig) error
}
