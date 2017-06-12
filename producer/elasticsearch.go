// Copyright 2015-2017 trivago GmbH
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
	"github.com/trivago/gollum/core"
	"sync"
)

// ElasticSearch producer plugin
//
// The ElasticSearch producer sends messages to elastic search using the bulk
// http API.
//
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
	core.BufferedProducer `gollumdoc:"embed_type"`
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
	core.TypeRegistry.Register(ElasticSearch{})
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
func (prod *ElasticSearch) Configure(conf core.PluginConfigReader) error {
	prod.BufferedProducer.Configure(conf)

	return conf.Errors.OrNil()
}

// Produce starts a bluk indexer
func (prod *ElasticSearch) Produce(workers *sync.WaitGroup) {

}
