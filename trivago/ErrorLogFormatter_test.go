package trivago

import (
	"encoding/json"
	"github.com/trivago/gollum/shared"
	"testing"
)

func expectMapping(t *testing.T, data map[string]interface{}, key string, value string) {
	val, valSet := data[key]
	if !valSet {
		t.Errorf("Expected key \"%s\" not found", key)
	}
	if val != value {
		t.Errorf("Expected \"%s\" for \"%s\" got \"%s\"", value, key, val)

	}
}

func TestErrorLogFormat(t *testing.T) {
	test := ErrorLogFormatter{}
	test.Configure(shared.PluginConfig{})
	msg := shared.NewMessage("www5-sfo.trivago.com [Thu Feb 12 11:57:27 2015] [error] [remote ip 10.11.2.52] [client 62.159.86.18] File does not exist: /appdata/www/trivago/autodiscover", []shared.MessageStreamID{})
	test.PrepareMessage(msg)

	parsedMsg := make(map[string]interface{})

	err := json.Unmarshal([]byte(test.String()), &parsedMsg)
	if err != nil {
		t.Error(err)
	}

	expectMapping(t, parsedMsg, "server", "www5-sfo.trivago.com")
	expectMapping(t, parsedMsg, "@timestamp", "2015-02-12T11:57:27Z")
	expectMapping(t, parsedMsg, "status", "error")
	expectMapping(t, parsedMsg, "remote", "ip 10.11.2.52")
	expectMapping(t, parsedMsg, "client", "62.159.86.18")
	expectMapping(t, parsedMsg, "message", "File does not exist: /appdata/www/trivago/autodiscover")
}
