// Copyright 2015-2016 trivago GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package producer

import (
	"bytes"
	elastigo "github.com/mattbaird/elastigo/lib"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo"
	"strconv"
	"sync"
	"time"
)

// ElasticSearch producer plugin
// The ElasticSearch producer sends messages to elastic search using the bulk
// http API. This producer uses a fuse breaker when cluster health reports a
// "red" status or the connection is down.
// Configuration example
//
//  - "producer.ElasticSearch":
//    Connections: 6
//    RetrySec: 5
//    TTL: ""
//    DayBasedIndex: false
//    User: ""
//    Password: ""
//    BatchSizeByte: 32768
//    BatchMaxCount: 256
//    BatchTimeoutSec: 5
//    Port: 9200
//    Servers:
//      - "localhost"
//    Index:
//      "console" : "console"
//      "_GOLLUM_"  : "_GOLLUM_"
//    Type:
//      "console" : "console"
//      "_GOLLUM_"  : "_GOLLUM_"
//
// RetrySec denotes the time in seconds after which a failed dataset will be
// transmitted again. By default this is set to 5.
//
// Connections defines the number of simultaneous connections allowed to a
// elasticsearch server. This is set to 6 by default.
//
// TTL defines the TTL set in elasticsearch messages. By default this is set to
// "" which means no TTL.
//
// DayBasedIndex can be set to true to append the date of the message to the
// index as in "<index>_YYYY-MM-DD". By default this is set to false.
//
// Servers defines a list of servers to connect to. The first server in the list
// is used as the server passed to the "Domain" setting. The Domain setting can
// be overwritten, too.
//
// Port defines the elasticsearch port, wich has to be the same for all servers.
// By default this is set to 9200.
//
// User and Password can be used to pass credentials to the elasticsearch server.
// By default both settings are empty.
//
// Index maps a stream to a specific index. You can define the
// wildcard stream (*) here, too. If set all streams that do not have a specific
// mapping will go to this stream (including _GOLLUM_).
// If no category mappings are set the stream name is used.
//
// Type maps a stream to a specific type. This behaves like the index map and
// is used to assign a _type to an elasticsearch message. By default the type
// "log" is used.
//
// BatchSizeByte defines the size in bytes required to trigger a flush.
// By default this is set to 32768 (32KB).
//
// BatchMaxCount defines the number of documents required to trigger a flush.
// By default this is set to 256.
//
// BatchTimeoutSec defines the time in seconds after which a flush will be
// triggered. By default this is set to 5.
type ElasticSearch struct {
	core.BufferedProducer
	conn          *elastigo.Conn
	indexer       *elastigo.BulkIndexer
	index         map[core.MessageStreamID]string
	msgType       map[core.MessageStreamID]string
	msgTTL        string
	dayBasedIndex bool
}

const (
	elasticMetricMessages    = "Elastic:Messages-"
	elasticMetricMessagesSec = "Elastic:MessagesSec-"
)

func init() {
	core.TypeRegistry.Register(ElasticSearch{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *ElasticSearch) Configure(conf core.PluginConfigReader) error {
	prod.BufferedProducer.Configure(conf)
	prod.SetStopCallback(prod.close)
	prod.SetCheckFuseCallback(prod.isClusterUp)

	defaultServer := []string{"localhost"}
	numConnections := conf.GetInt("Connections", 6)
	retrySec := conf.GetInt("RetrySec", 5)

	prod.conn = elastigo.NewConn()
	prod.conn.Hosts = conf.GetStringArray("Servers", defaultServer)
	prod.conn.Domain = conf.GetString("Domain", prod.conn.Hosts[0])
	prod.conn.ClusterDomains = prod.conn.Hosts
	prod.conn.Port = strconv.Itoa(conf.GetInt("Port", 9200))
	prod.conn.Username = conf.GetString("User", "")
	prod.conn.Password = conf.GetString("Password", "")

	prod.indexer = prod.conn.NewBulkIndexerErrors(numConnections, retrySec)
	prod.indexer.BufferDelayMax = time.Duration(conf.GetInt("BatchTimeoutSec", 5)) * time.Second
	prod.indexer.BulkMaxBuffer = conf.GetInt("BatchSizeByte", 32768)
	prod.indexer.BulkMaxDocs = conf.GetInt("BatchMaxCount", 128)

	prod.indexer.Sender = func(buf *bytes.Buffer) error {
		_, err := prod.conn.DoCommand("POST", "/_bulk", nil, buf)
		return err
	}

	prod.index = conf.GetStreamMap("Index", "")
	prod.msgType = conf.GetStreamMap("Type", "log")
	prod.msgTTL = conf.GetString("TTL", "")
	prod.dayBasedIndex = conf.GetBool("DayBasedIndex", false)

	for _, index := range prod.index {
		metricName := elasticMetricMessages + index
		tgo.Metric.New(metricName)
		tgo.Metric.NewRate(metricName, elasticMetricMessagesSec+index, time.Second, 10, 3, true)
	}

	return conf.Errors.OrNil()
}

func (prod *ElasticSearch) isClusterUp() bool {
	cluster, err := prod.conn.Health()
	if err != nil {
		return false
	}

	return !cluster.TimedOut && cluster.Status != "red"
}

func (prod *ElasticSearch) sendMessage(msg *core.Message) {
	originalMsg := *msg
	prod.BufferedProducer.Format(msg)

	index, indexMapped := prod.index[msg.StreamID()]
	if !indexMapped {
		index, indexMapped = prod.index[core.WildcardStreamID]
		if !indexMapped {
			index = core.StreamRegistry.GetStreamName(msg.StreamID())
		}
		metricName := elasticMetricMessages + index
		tgo.Metric.New(metricName)
		tgo.Metric.NewRate(metricName, elasticMetricMessagesSec+index, time.Second, 10, 3, true)
		prod.index[msg.StreamID()] = index
	}

	if prod.dayBasedIndex {
		index = index + "_" + msg.Created().Format("2006-01-02")
	}

	msgType, typeMapped := prod.msgType[msg.StreamID()]
	if !typeMapped {
		msgType, typeMapped = prod.msgType[core.WildcardStreamID]
		if !typeMapped {
			msgType = core.StreamRegistry.GetStreamName(msg.StreamID())
		}
	}

	timestamp := msg.Created()
	err := prod.indexer.Index(index, msgType, "", "", prod.msgTTL, &timestamp, msg.String())
	if err != nil {
		prod.Log.Error.Print("ElasticSearch index error - ", err)
		if !prod.isClusterUp() {
			prod.Control() <- core.PluginControlFuseBurn
		}
		prod.Drop(&originalMsg)
	} else {
		tgo.Metric.Inc(elasticMetricMessages + index)
		prod.Control() <- core.PluginControlFuseActive
	}
}

func (prod *ElasticSearch) close() {
	defer prod.WorkerDone()
	prod.DefaultClose()
	prod.indexer.Flush()
	prod.indexer.Stop()
}

// Produce starts a bluk indexer
func (prod *ElasticSearch) Produce(workers *sync.WaitGroup) {
	prod.indexer.Start()
	prod.AddMainWorker(workers)
	prod.MessageControlLoop(prod.sendMessage)
}
