package trivago

import (
	"encoding/json"
	"github.com/trivago/gollum/shared"
	"testing"
)

func TestErrorLogFormat(t *testing.T) {
	expect := shared.NewExpect(t)

	test := ErrorLogFormatter{}
	test.Configure(shared.PluginConfig{})
	msg := shared.NewMessage("www5-sfo.trivago.com [Thu Feb 12 11:57:27 2015] [error] [remote ip 10.11.2.52] [client 62.159.86.18] File does not exist: /appdata/www/trivago/autodiscover", []shared.MessageStreamID{})
	test.PrepareMessage(msg)

	parsedMsg := make(map[string]interface{})

	err := json.Unmarshal([]byte(test.String()), &parsedMsg)
	if err != nil {
		t.Error(err)
	}

	expect.MapSetEq(parsedMsg, "server", "www5-sfo.trivago.com")
	expect.MapSetEq(parsedMsg, "@timestamp", "2015-02-12T11:57:27Z")
	expect.MapSetEq(parsedMsg, "status", "error")
	expect.MapSetEq(parsedMsg, "remote", "ip 10.11.2.52")
	expect.MapSetEq(parsedMsg, "client", "62.159.86.18")
	expect.MapSetEq(parsedMsg, "message", "File does not exist: /appdata/www/trivago/autodiscover")
}
