package trivago

import (
	"encoding/json"
	"github.com/trivago/gollum/shared"
	"testing"
)

func TestAccessLogFormat1(t *testing.T) {
	expect := shared.NewExpect(t)

	test := AccessLogFormatter{}
	test.Configure(shared.PluginConfig{})
	msg := shared.NewMessage("10.1.2.44 www.trivago.de 93.222.62.57 - 20150212144115 \"GET /widgets/dealform?w_cip=90006200&w_theme=theme_id&w_style=darkblue&w_path=&w_item=0&w_size=7 HTTP/1.1\" 301 275 102909 WidgetBundle_DealformWidget_load 0 0 1 6 0 0 1 \"http://www.wetter.com/wetter_aktuell/wettervorhersage/16_tagesvorhersage/?id=DE0000110\" \"VNytewoBAiwAARwFYDYAAABj\" \"Mozilla/5.0 (Windows NT 6.1; WOW64; Trident/7.0; rv:11.0) like Gecko\"", []shared.MessageStreamID{})
	test.PrepareMessage(msg)

	parsedMsg := make(map[string]interface{})

	err := json.Unmarshal([]byte(test.String()), &parsedMsg)
	if err != nil {
		t.Error(err)
	}

	expect.MapSetEq(parsedMsg, "serverIP", "10.1.2.44")
	expect.MapSetEq(parsedMsg, "serverName", "www.trivago.de")
	expect.MapSetEq(parsedMsg, "forwardedFor", "93.222.62.57")
	expect.MapNotSet(parsedMsg, "remoteIP")
	expect.MapSetEq(parsedMsg, "@timestamp", "2015-02-12T14:41:15Z")
	expect.MapSetEq(parsedMsg, "method", "GET")
	expect.MapSetEq(parsedMsg, "request", "/widgets/dealform?w_cip=90006200&w_theme=theme_id&w_style=darkblue&w_path=&w_item=0&w_size=7")
	expect.MapSetEq(parsedMsg, "protocol", "HTTP/1.1")
	expect.MapSetEq(parsedMsg, "status", "301")
	expect.MapSetEq(parsedMsg, "responseSize", "275")
	expect.MapSetEq(parsedMsg, "responseTime", "102909")
	expect.MapSetEq(parsedMsg, "pageID", "WidgetBundle_DealformWidget_load")
	expect.MapSetEq(parsedMsg, "metricDb", "0")
	expect.MapSetEq(parsedMsg, "metricJava", "0")
	expect.MapSetEq(parsedMsg, "metricMemcache", "1")
	expect.MapSetEq(parsedMsg, "metricXCache", "6")
	expect.MapSetEq(parsedMsg, "metricFTP", "0")
	expect.MapSetEq(parsedMsg, "metricSolr", "0")
	expect.MapSetEq(parsedMsg, "metricRedis", "1")
	expect.MapSetEq(parsedMsg, "referrer", "http://www.wetter.com/wetter_aktuell/wettervorhersage/16_tagesvorhersage/?id=DE0000110")
	expect.MapSetEq(parsedMsg, "hash", "VNytewoBAiwAARwFYDYAAABj")
	expect.MapSetEq(parsedMsg, "agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; Trident/7.0; rv:11.0) like Gecko")
}

func TestAccessLogFormat2(t *testing.T) {
	expect := shared.NewExpect(t)

	test := AccessLogFormatter{}
	test.Configure(shared.PluginConfig{})
	msg := shared.NewMessage("10.1.2.44 www.trivago.com - - 20150212144115 \"OPTIONS /lbcheck.php HTTP/1.0\" 200 15 609 - - - - - - - - \"-\" \"VNytewoBAiwAARwFYDgAAABj\" \"-\"", []shared.MessageStreamID{})
	test.PrepareMessage(msg)

	parsedMsg := make(map[string]interface{})

	err := json.Unmarshal([]byte(test.String()), &parsedMsg)
	if err != nil {
		t.Error(err)
	}

	expect.MapSetEq(parsedMsg, "serverIP", "10.1.2.44")
	expect.MapSetEq(parsedMsg, "serverName", "www.trivago.com")
	expect.MapNotSet(parsedMsg, "forwardedFor")
	expect.MapNotSet(parsedMsg, "remoteIP")
	expect.MapSetEq(parsedMsg, "@timestamp", "2015-02-12T14:41:15Z")
	expect.MapSetEq(parsedMsg, "method", "OPTIONS")
	expect.MapSetEq(parsedMsg, "request", "/lbcheck.php")
	expect.MapSetEq(parsedMsg, "protocol", "HTTP/1.0")
	expect.MapSetEq(parsedMsg, "status", "200")
	expect.MapSetEq(parsedMsg, "responseSize", "15")
	expect.MapSetEq(parsedMsg, "responseTime", "609")
	expect.MapNotSet(parsedMsg, "pageID")
	expect.MapNotSet(parsedMsg, "metricDb")
	expect.MapNotSet(parsedMsg, "metricJava")
	expect.MapNotSet(parsedMsg, "metricMemcache")
	expect.MapNotSet(parsedMsg, "metricXCache")
	expect.MapNotSet(parsedMsg, "metricFTP")
	expect.MapNotSet(parsedMsg, "metricSolr")
	expect.MapNotSet(parsedMsg, "metricRedis")
	expect.MapNotSet(parsedMsg, "referrer")
	expect.MapSetEq(parsedMsg, "hash", "VNytewoBAiwAARwFYDgAAABj")
	expect.MapNotSet(parsedMsg, "agent")
}

func TestAccessLogFormat3(t *testing.T) {
	expect := shared.NewExpect(t)

	test := AccessLogFormatter{}
	test.Configure(shared.PluginConfig{})
	msg := shared.NewMessage("10.1.2.44 www.trivago.de 79.236.28.139, 10.1.2.81 - 20150212144115 \"GET /search/de-DE-DE/v9_02_1_ae_cache/suggest?q=bad HTTP/1.1\" 200 537 99233 PriceSearchBundle_suggest 0 0 1 6 0 0 0 \"http://www.trivago.de/?iSemThemeId=17122&iPathId=655&bDispMoreFilter=false&aDateRange%5Barr%5D=2015-03-08&aDateRange%5Bdep%5D=2015-03-09&aCategoryRange=0%2C1%2C2%2C3%2C4%2C5&iRoomType=7&sOrderBy=relevance%20desc&aPartner=&aOverallLiking=1%2C2%2C3%2C4%2C5&iOffset=50&iLimit=25&iIncludeAll=0&bTopDealsOnly=true&iViewType=0&aPriceRange%5Bfrom%5D=0&aPriceRange%5Bto%5D=0&aGeoCode%5Blng%5D=10.451526&aGeoCode%5Blat%5D=51.165691&bIsSeoPage=false&mgo=false&bHotelTestContext=false&th=false&aHotelTestClassifier=&bSharedRooms=false&bIsTotalPrice=false&bIsSitemap=false&rp=&sSemKeywordInfo=last+minute+hotel+deutschland&ww=false&\" \"VNytewoBAiwAAUladOYAAAAK\" \"Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/40.0.2214.111 Safari/537.36\"", []shared.MessageStreamID{})
	test.PrepareMessage(msg)

	parsedMsg := make(map[string]interface{})

	err := json.Unmarshal([]byte(test.String()), &parsedMsg)
	if err != nil {
		t.Error(err)
	}

	expect.MapSetEq(parsedMsg, "serverIP", "10.1.2.44")
	expect.MapSetEq(parsedMsg, "serverName", "www.trivago.de")
	expect.MapSetEq(parsedMsg, "forwardedFor", "79.236.28.139, 10.1.2.81")
	expect.MapNotSet(parsedMsg, "remoteIP")
	expect.MapSetEq(parsedMsg, "@timestamp", "2015-02-12T14:41:15Z")
	expect.MapSetEq(parsedMsg, "method", "GET")
	expect.MapSetEq(parsedMsg, "request", "/search/de-DE-DE/v9_02_1_ae_cache/suggest?q=bad")
	expect.MapSetEq(parsedMsg, "protocol", "HTTP/1.1")
	expect.MapSetEq(parsedMsg, "status", "200")
	expect.MapSetEq(parsedMsg, "responseSize", "537")
	expect.MapSetEq(parsedMsg, "responseTime", "99233")
	expect.MapSetEq(parsedMsg, "pageID", "PriceSearchBundle_suggest")
	expect.MapSetEq(parsedMsg, "metricDb", "0")
	expect.MapSetEq(parsedMsg, "metricJava", "0")
	expect.MapSetEq(parsedMsg, "metricMemcache", "1")
	expect.MapSetEq(parsedMsg, "metricXCache", "6")
	expect.MapSetEq(parsedMsg, "metricFTP", "0")
	expect.MapSetEq(parsedMsg, "metricSolr", "0")
	expect.MapSetEq(parsedMsg, "metricRedis", "0")
	expect.MapSetEq(parsedMsg, "referrer", "http://www.trivago.de/?iSemThemeId=17122&iPathId=655&bDispMoreFilter=false&aDateRange%5Barr%5D=2015-03-08&aDateRange%5Bdep%5D=2015-03-09&aCategoryRange=0%2C1%2C2%2C3%2C4%2C5&iRoomType=7&sOrderBy=relevance%20desc&aPartner=&aOverallLiking=1%2C2%2C3%2C4%2C5&iOffset=50&iLimit=25&iIncludeAll=0&bTopDealsOnly=true&iViewType=0&aPriceRange%5Bfrom%5D=0&aPriceRange%5Bto%5D=0&aGeoCode%5Blng%5D=10.451526&aGeoCode%5Blat%5D=51.165691&bIsSeoPage=false&mgo=false&bHotelTestContext=false&th=false&aHotelTestClassifier=&bSharedRooms=false&bIsTotalPrice=false&bIsSitemap=false&rp=&sSemKeywordInfo=last+minute+hotel+deutschland&ww=false&")
	expect.MapSetEq(parsedMsg, "hash", "VNytewoBAiwAAUladOYAAAAK")
	expect.MapSetEq(parsedMsg, "agent", "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/40.0.2214.111 Safari/537.36")
}
