package tests

import (
	"encoding/json"
	"github.com/trivago/gollum/shared"
	"github.com/trivago/gollum/trivago"
	"testing"
)

func TestAccessLogFormat1(t *testing.T) {
	test := trivago.AccessLogFormatter{}
	test.Configure(shared.PluginConfig{})
	msg := shared.NewMessage("10.1.2.44 www.trivago.de 93.222.62.57 - 20150212144115 \"GET /widgets/dealform?w_cip=90006200&w_theme=theme_id&w_style=darkblue&w_path=&w_item=0&w_size=7 HTTP/1.1\" 301 275 102909 WidgetBundle_DealformWidget_load 0 0 1 6 0 0 1 \"http://www.wetter.com/wetter_aktuell/wettervorhersage/16_tagesvorhersage/?id=DE0000110\" \"VNytewoBAiwAARwFYDYAAABj\" \"Mozilla/5.0 (Windows NT 6.1; WOW64; Trident/7.0; rv:11.0) like Gecko\"", []shared.MessageStreamID{})
	test.PrepareMessage(msg)

	parsedMsg := make(map[string]interface{})

	err := json.Unmarshal([]byte(test.String()), &parsedMsg)
	if err != nil {
		t.Error(err)
	}

	expectMapping(t, parsedMsg, "serverIP", "10.1.2.44")
	expectMapping(t, parsedMsg, "serverName", "www.trivago.de")
	expectMapping(t, parsedMsg, "clientIP", "93.222.62.57")
	expectMapping(t, parsedMsg, "clientIP3", "-")
	expectMapping(t, parsedMsg, "@timestamp", "2015-02-12T14:41:15Z")
	expectMapping(t, parsedMsg, "method", "GET")
	expectMapping(t, parsedMsg, "request", "/widgets/dealform?w_cip=90006200&w_theme=theme_id&w_style=darkblue&w_path=&w_item=0&w_size=7")
	expectMapping(t, parsedMsg, "protocol", "HTTP/1.1")
	expectMapping(t, parsedMsg, "resultCode", "301")
	expectMapping(t, parsedMsg, "requestSize", "275")
	expectMapping(t, parsedMsg, "responseSize", "102909")
	expectMapping(t, parsedMsg, "command", "WidgetBundle_DealformWidget_load")
	expectMapping(t, parsedMsg, "flags", "0 0 1 6 0 0 1")
	expectMapping(t, parsedMsg, "referrer", "http://www.wetter.com/wetter_aktuell/wettervorhersage/16_tagesvorhersage/?id=DE0000110")
	expectMapping(t, parsedMsg, "session", "VNytewoBAiwAARwFYDYAAABj")
	expectMapping(t, parsedMsg, "agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; Trident/7.0; rv:11.0) like Gecko")
}

func TestAccessLogFormat2(t *testing.T) {
	test := trivago.AccessLogFormatter{}
	test.Configure(shared.PluginConfig{})
	msg := shared.NewMessage("10.1.2.44 www.trivago.com - - 20150212144115 \"OPTIONS /lbcheck.php HTTP/1.0\" 200 15 609 - - - - - - - - \"-\" \"VNytewoBAiwAARwFYDgAAABj\" \"-\"", []shared.MessageStreamID{})
	test.PrepareMessage(msg)

	parsedMsg := make(map[string]interface{})

	err := json.Unmarshal([]byte(test.String()), &parsedMsg)
	if err != nil {
		t.Error(err)
	}

	expectMapping(t, parsedMsg, "serverIP", "10.1.2.44")
	expectMapping(t, parsedMsg, "serverName", "www.trivago.com")
	expectMapping(t, parsedMsg, "clientIP", "-")
	expectMapping(t, parsedMsg, "clientIP3", "-")
	expectMapping(t, parsedMsg, "@timestamp", "2015-02-12T14:41:15Z")
	expectMapping(t, parsedMsg, "method", "OPTIONS")
	expectMapping(t, parsedMsg, "request", "/lbcheck.php")
	expectMapping(t, parsedMsg, "protocol", "HTTP/1.0")
	expectMapping(t, parsedMsg, "resultCode", "200")
	expectMapping(t, parsedMsg, "requestSize", "15")
	expectMapping(t, parsedMsg, "responseSize", "609")
	expectMapping(t, parsedMsg, "command", "-")
	expectMapping(t, parsedMsg, "flags", "- - - - - - -")
	expectMapping(t, parsedMsg, "referrer", "-")
	expectMapping(t, parsedMsg, "session", "VNytewoBAiwAARwFYDgAAABj")
	expectMapping(t, parsedMsg, "agent", "-")
}
