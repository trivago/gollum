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
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	"strconv"
	"sync"
	"sync/atomic"
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
//    Settings:
//      "console":
//        "number_of_shards": 1
//    DataTypes:
//      "console":
//        "source": "ip"
//    Type:
//      "console" : "log"
//      "_GOLLUM_"  : "log"
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
// Port defines the elasticsearch port, which has to be the same for all servers.
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
// DataTypes allows to define elasticsearch type mappings for indexes that are
// being created by this producer (e.g. day based indexes). You can define
// mappings per index.
//
// Settings allows to define elasticsearch index settings for indexes that are
// being created by this producer (e.g. day based indexes). You can define
// settings per index.
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
	core.ProducerBase
	conn               *elastigo.Conn
	indexer            *elastigo.BulkIndexer
	index              map[core.MessageStreamID]string
	msgType            map[core.MessageStreamID]string
	msgTTL             string
	dayBasedIndex      bool
	counters           map[string]*int64
	indexSettings      map[string]*elasticSettings
	indexSettingsSent  map[string]string
	indexSettingsGuard *sync.Mutex
	lastMetricUpdate   time.Time
}

type elasticType struct {
	TypeName interface{} `json:"type"`
}

type elasticMapping struct {
	Properties map[string]elasticType `json:"properties,omitempty"`
}

type elasticSettings struct {
	Settings map[string]interface{}    `json:"settings,omitempty"`
	Mappings map[string]elasticMapping `json:"mappings,omitempty"`
}

const (
	elasticMetricMessages    = "Elastic:Messages-"
	elasticMetricMessagesSec = "Elastic:MessagesSec-"
)

func init() {
	shared.TypeRegistry.Register(ElasticSearch{})
}

func makeElasticSettings() *elasticSettings {
	return &elasticSettings{
		Settings: make(map[string]interface{}),
		Mappings: make(map[string]elasticMapping),
	}
}

func makeElasticMapping() elasticMapping {
	return elasticMapping{
		Properties: make(map[string]elasticType),
	}
}

// Configure initializes this producer with values from a plugin config.
func (prod *ElasticSearch) Configure(conf core.PluginConfig) error {
	err := prod.ProducerBase.Configure(conf)
	if err != nil {
		return err
	}

	prod.SetStopCallback(prod.close)

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
		if err != nil {
			Log.Error.Print("ElasticSearch response error - ", err)
		}
		return err
	}

	prod.index = conf.GetStreamMap("Index", "")
	prod.msgType = conf.GetStreamMap("Type", "log")
	prod.msgTTL = conf.GetString("TTL", "")
	prod.dayBasedIndex = conf.GetBool("DayBasedIndex", false)

	prod.counters = make(map[string]*int64)
	prod.lastMetricUpdate = time.Now()

	for _, index := range prod.index {
		shared.Metric.New(elasticMetricMessages + index)
		shared.Metric.New(elasticMetricMessagesSec + index)
		prod.counters[index] = new(int64)
	}

	prod.indexSettingsGuard = new(sync.Mutex)
	prod.indexSettingsSent = make(map[string]string)
	prod.indexSettings = make(map[string]*elasticSettings)

	conf.HasValue("DataTypes")
	indexTypes, err := conf.Settings.MarshalMap("DataTypes")
	if indexTypes != nil {
		for index := range indexTypes {
			mappings := makeElasticMapping()
			types, _ := indexTypes.MarshalMap(index)
			for name := range types {
				typeName, _ := types.String(name)
				mappings.Properties[name] = elasticType{TypeName: typeName}
			}

			indexType := prod.getIndexType(core.StreamRegistry.GetStreamID(index))
			settings := makeElasticSettings()
			settings.Mappings[indexType] = mappings
			prod.indexSettings[index] = settings
		}
	}

	conf.HasValue("Settings")
	indexSettings, _ := conf.Settings.MarshalMap("Settings")
	if indexSettings != nil {
		for index := range indexSettings {
			mappings, exist := prod.indexSettings[index]
			if !exist {
				mappings = makeElasticSettings()
				prod.indexSettings[index] = mappings
			}
			mappings.Settings, _ = indexSettings.MarshalMap(index)
		}
	}

	prod.SetCheckFuseCallback(prod.isClusterUp)
	return nil
}

func (prod *ElasticSearch) updateMetrics() {
	duration := time.Since(prod.lastMetricUpdate)
	prod.lastMetricUpdate = time.Now()

	for index, counter := range prod.counters {
		count := atomic.SwapInt64(counter, 0)
		shared.Metric.Add(elasticMetricMessages+index, count)
		shared.Metric.SetF(elasticMetricMessagesSec+index, float64(count)/duration.Seconds())
	}
}

func (prod *ElasticSearch) isClusterUp() bool {
	cluster, err := prod.conn.Health()
	if err != nil {
		return false
	}

	return !cluster.TimedOut && cluster.Status != "red"
}

func (prod *ElasticSearch) createIndex(index string, sendIndex string, indexType string) {
	prod.indexSettingsGuard.Lock()
	defer prod.indexSettingsGuard.Unlock()
	if value, isSet := prod.indexSettingsSent[index]; isSet && value == sendIndex {
		return
	}
	if settings, exist := prod.indexSettings[index]; exist {
		if onServer, _ := prod.conn.ExistsIndex(sendIndex, indexType, make(map[string]interface{})); !onServer {
			if _, err := prod.conn.CreateIndexWithSettings(sendIndex, *settings); err != nil {
				Log.Error.Print("ElasticSearch index creation error - ", err.Error())
			}
		}
	}
	prod.indexSettingsSent[index] = sendIndex
}

func (prod *ElasticSearch) getIndexType(streamID core.MessageStreamID) string {
	indexType, typeMapped := prod.msgType[streamID]
	if typeMapped {
		return indexType
	}
	indexType, typeMapped = prod.msgType[core.WildcardStreamID]
	if typeMapped {
		return indexType
	}
	return core.StreamRegistry.GetStreamName(streamID)
}

func (prod *ElasticSearch) sendMessage(msg core.Message) {
	originalMsg := msg
	msg.Data, msg.StreamID = prod.ProducerBase.Format(msg)

	index, indexMapped := prod.index[msg.StreamID]
	if !indexMapped {
		index, indexMapped = prod.index[core.WildcardStreamID]
		if !indexMapped {
			index = core.StreamRegistry.GetStreamName(msg.StreamID)
		}
		shared.Metric.New(elasticMetricMessages + index)
		shared.Metric.New(elasticMetricMessagesSec + index)
		prod.counters[index] = new(int64)
		prod.index[msg.StreamID] = index
	}

	indexType := prod.getIndexType(msg.StreamID)
	atomic.AddInt64(prod.counters[index], 1)

	sendIndex := index
	if prod.dayBasedIndex {
		sendIndex += msg.Timestamp.Format("_2006-01-02")
	}
	prod.createIndex(index, sendIndex, indexType)

	err := prod.indexer.Index(sendIndex, indexType, "", "", prod.msgTTL, &msg.Timestamp, string(msg.Data))
	if err != nil {
		Log.Error.Print("ElasticSearch index error - ", err)
		if !prod.isClusterUp() {
			prod.Control() <- core.PluginControlFuseBurn
		}
		prod.Drop(originalMsg)
	} else {
		prod.Control() <- core.PluginControlFuseActive
	}
}

func (prod *ElasticSearch) close() {
	defer prod.WorkerDone()
	prod.CloseMessageChannel(prod.sendMessage)
	prod.indexer.Flush()
	prod.indexer.Stop()
}

// Produce starts a bluk indexer
func (prod *ElasticSearch) Produce(workers *sync.WaitGroup) {
	prod.indexer.Start()
	prod.AddMainWorker(workers)
	prod.TickerMessageControlLoop(prod.sendMessage, time.Second*5, prod.updateMetrics)
}
