package trivago

import (
	"bytes"
	"github.com/trivago/gollum/shared"
	"time"
)

type acclTransition struct {
	character rune
	state     int
}

const (
	acclStateServerIP = iota
	acclStateServerName
	acclStateForwardedFor
	acclStateRemoteIP
	acclStateTimestamp
	acclStateMethod
	acclStateRequest
	acclStateProtocol
	acclStateStatus
	acclStateResponseSize
	acclStateResponseTime
	acclStatePageID
	acclStateMetricDb
	acclStateMetricJava
	acclStateMetricMemcache
	acclStateMetricXCache
	acclStateMetricFTP
	acclStateMetricSolr
	acclStateMetricRedis
	acclStateReferrer
	acclStateHash
	acclStateAgent
)

var acclStateNames = []string{
	"serverIP",
	"serverName",
	"forwardedFor",
	"remoteIP",
	"@timestamp",
	"method",
	"request",
	"protocol",
	"status",
	"responseSize",
	"responseTime",
	"pageID",
	"metricDb",
	"metricJava",
	"metricMemcache",
	"metricXCache",
	"metricFTP",
	"metricSolr",
	"metricRedis",
	"referrer",
	"hash",
	"agent",
}

var acclTransitions = [][]shared.Transition{
	/* serverIP       */ {transWrite(" ", acclStateServerName)},
	/* serverName     */ {transWrite(" ", acclStateForwardedFor)},
	/* ForwardedFor   */ {transIgnore(", ", acclStateForwardedFor), transWrite(" ", acclStateRemoteIP), transSkip("- ", acclStateRemoteIP)},
	/* remoteIP       */ {transWrite(" ", acclStateTimestamp), transSkip("- ", acclStateTimestamp)},
	/* timestamp      */ {transWrite(" \"", acclStateMethod)},
	/* method         */ {transWrite(" ", acclStateRequest)},
	/* request        */ {transWrite(" ", acclStateProtocol)},
	/* protocol       */ {transWrite("\" ", acclStateStatus)},
	/* status         */ {transWrite(" ", acclStateResponseSize)},
	/* responseSize   */ {transWrite(" ", acclStateResponseTime)},
	/* responseTime   */ {transWrite(" ", acclStatePageID)},
	/* pageID         */ {transWrite(" ", acclStateMetricDb), transSkip("- ", acclStateMetricDb)},
	/* metricDb       */ {transWrite(" ", acclStateMetricJava), transSkip("- ", acclStateMetricJava)},
	/* metricJava     */ {transWrite(" ", acclStateMetricMemcache), transSkip("- ", acclStateMetricMemcache)},
	/* metricMemcache */ {transWrite(" ", acclStateMetricXCache), transSkip("- ", acclStateMetricXCache)},
	/* metricXCache   */ {transWrite(" ", acclStateMetricFTP), transSkip("- ", acclStateMetricFTP)},
	/* metricFTP      */ {transWrite(" ", acclStateMetricSolr), transSkip("- ", acclStateMetricSolr)},
	/* metricSolr     */ {transWrite(" ", acclStateMetricRedis), transSkip("- ", acclStateMetricRedis)},
	/* metricRedis    */ {transWrite(" \"", acclStateReferrer), transSkip("- \"", acclStateReferrer)},
	/* referrer       */ {transWrite("\" \"", acclStateHash), transSkip("-\" \"", acclStateHash)},
	/* hash           */ {transWrite("\" \"", acclStateAgent)},
	/* agent          */ {transWrite("\"", shared.ParserStateStop), transSkip("-\"", shared.ParserStateStop)},
}

type AccessLogFormatter struct {
	JSONLogFormatter
}

func init() {
	shared.RuntimeType.Register(AccessLogFormatter{})
}

func (format *AccessLogFormatter) Configure(conf shared.PluginConfig) error {
	format.parser = shared.NewParser(acclTransitions)
	return nil
}

func (format *AccessLogFormatter) PrepareMessage(msg shared.Message) {
	sections := format.parser.Parse([]byte(msg.Data), acclStateServerIP)
	isFirst := true
	format.message = bytes.NewBufferString("{")

	for _, section := range sections {
		switch section.State {
		case acclStateTimestamp:
			timeStamp, _ := time.Parse("20060102150405", string(bytes.TrimSpace(section.Data)))
			format.writeField("@timestamp", []byte(timeStamp.Format(time.RFC3339)), &isFirst)

		default:
			format.writeField(acclStateNames[section.State], section.Data, &isFirst)
		}
	}

	format.message.WriteString("}")
}
