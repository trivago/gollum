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
	"context"
	"fmt"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tcontainer"
	"gopkg.in/olivere/elastic.v5"
	"sync"
)

// ElasticSearch producer plugin
//
// The ElasticSearch producer sends messages to elastic search using the bulk
// http API. The producer expects a json payload.
//
// Configuration example
//
//  producerElasticSearch:
// 	  Type: producer.ElasticSearch
//    Connections: 6
//    #RetrySec: 5
//    TTL: ""
//    DayBasedIndex: false
//    User: ""
//    Password: ""
//    BatchSizeByte: 32768
//    BatchMaxCount: 256
//    BatchTimeoutSec: 5
//    Servers:
//      - http://127.0.0.1:9200
//    StreamProperties:
//		write:
//			Index: twitter
//			Type: tweet
//
// 			# see: https://www.elastic.co/guide/en/elasticsearch/reference/5.4/indices-create-index.html#mappings
//			Mapping:
//				# index mapping for payload
// 				user: keyword
//				message: text
//			Settings:
//				# settings used for mapping
//				number_of_shards: 5
//				number_of_replicas: 1
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
// Servers defines a list of servers to connect to.
//
// User and Password can be used to pass credentials to the elasticsearch server.
// By default both settings are empty.
//
// StreamProperties defines the mapping and settings for each stream.
// As index use the stream name here.
//
// StreamProperties/<streamName>/Index
// Elasticsearch index which used for the stream.
//
// StreamProperties/<streamName>/Type
// Document type which used for the stream.
//
// StreamProperties/<streamName>/Mapping is a map which used for the document field mapping.
// As document type the already definded type is reused for the field mapping
// See https://www.elastic.co/guide/en/elasticsearch/reference/5.4/indices-create-index.html#mappings
//
// StreamProperties/<streamName>/Settings is a map which is used for the index settings.
// See https://www.elastic.co/guide/en/elasticsearch/reference/5.4/indices-create-index.html#mappings
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
	connection            elasticConnection

	indexSettings map[string]*elasticIndex

	indexMap map[core.MessageStreamID]string
	typeMap  map[core.MessageStreamID]string
}

type elasticConnection struct {
	servers           []string
	user              string
	password          string
	client            *elastic.Client
	isConnectedStatus bool
}

func (conn *elasticConnection) isConnected() bool {
	return conn.isConnectedStatus
}

func (conn *elasticConnection) connect() error {
	conf := []elastic.ClientOptionFunc{elastic.SetURL(conn.servers...), elastic.SetSniff(false)}
	if len(conn.user) > 0 {
		conf = append(conf, elastic.SetBasicAuth(conn.user, conn.password))
	}

	client, err := elastic.NewClient(conf...)
	if err != nil {
		return err
	}

	conn.client = client
	conn.isConnectedStatus = true

	return nil
}

type elasticIndex struct {
	Settings map[string]interface{}    `json:"settings,omitempty"`
	Mappings map[string]elasticMapping `json:"mappings,omitempty"`
}

type elasticMapping struct {
	Properties map[string]elasticType `json:"properties,omitempty"`
}

type elasticType struct {
	TypeName interface{} `json:"type"`
}

func init() {
	core.TypeRegistry.Register(ElasticSearch{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *ElasticSearch) Configure(conf core.PluginConfigReader) error {
	prod.BufferedProducer.Configure(conf)

	prod.connection.servers = conf.GetStringArray("Servers", []string{"http://127.0.0.1:9200"})
	prod.connection.user = conf.GetString("User", "")
	prod.connection.password = conf.GetString("Password", "")
	prod.connection.isConnectedStatus = false

	prod.configureIndexSettings(conf.GetMap("StreamProperties", tcontainer.NewMarshalMap()))

	return conf.Errors.OrNil()
}

func (prod *ElasticSearch) configureIndexSettings(properties tcontainer.MarshalMap) {
	prod.indexSettings = map[string]*elasticIndex{}
	prod.indexMap = map[core.MessageStreamID]string{}
	prod.typeMap = map[core.MessageStreamID]string{}

	if len(properties) <= 0 {
		prod.Log.Error.Print("No stream configuration found. Please check your config.")
		return
	}

	for streamName := range properties {
		streamID := core.GetStreamID(streamName)

		property, err := properties.MarshalMap(streamName)
		if err != nil {
			prod.Log.Error.Printf("no configuration found for stream '%s'. Please check your config.", streamName)
		}

		indexName, err := property.String("index")
		if err != nil {
			prod.Log.Error.Printf("no index configured for stream '%s'. Please check your config.", streamName)
		}

		elType, err := property.String("type")
		if err != nil {
			prod.Log.Error.Printf("no data type configured for stream '%s'. Please check your config.", streamName)
		}

		// set class properties
		prod.indexSettings[indexName] = prod.newElasticIndex(property)

		prod.typeMap[streamID] = elType
		prod.indexMap[streamID] = indexName
	}
}

func (prod *ElasticSearch) newElasticIndex(property tcontainer.MarshalMap) *elasticIndex {
	elType, _ := property.String("type")
	mapping, _ := property.MarshalMap("mapping")
	settings, _ := property.MarshalMap("settings")

	elMappings := elasticMapping{
		Properties: make(map[string]elasticType),
	}

	for fieldName := range mapping {
		typeName, _ := mapping.String(fieldName)
		elMappings.Properties[fieldName] = elasticType{TypeName: typeName}
	}

	// init elasticIndex instance
	elIndex := elasticIndex{
		Settings: make(map[string]interface{}),
		Mappings: make(map[string]elasticMapping),
	}

	elIndex.Mappings[elType] = elMappings
	elIndex.Settings = settings

	return &elIndex
}

// Produce starts the producer
func (prod *ElasticSearch) Produce(workers *sync.WaitGroup) {
	prod.initIndex()
	prod.AddMainWorker(workers)
	prod.MessageControlLoop(prod.sendMessage)
}

func (prod *ElasticSearch) getClient() (*elastic.Client, error) {
	if !prod.connection.isConnected() {
		err := prod.connection.connect()
		if err != nil {
			prod.Log.Error.Print("Error during connection: ", err)
			return nil, err
		}
	}

	return prod.connection.client, nil
}

func (prod *ElasticSearch) initIndex() {
	for indexName, elIndex := range prod.indexSettings {
		prod.createIndex(indexName, elIndex)
	}
}

func (prod *ElasticSearch) createIndex(indexName string, elIndex *elasticIndex) error {
	client, err := prod.getClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	exists, err := client.IndexExists(indexName).Do(ctx)
	if err != nil {
		prod.Log.Error.Println("Issue during checking index: ", err)
		return err
	}
	if !exists {
		_, err = client.CreateIndex(indexName).Do(ctx)
		if err != nil {
			prod.Log.Error.Println("Issue during creating index: ", err)
			return err
		}

		prod.Log.Debug.Printf("Created index '%s'\n", indexName)
	}

	for typeName, properties := range elIndex.Mappings {
		mapping := client.PutMapping()
		mapping.Index(indexName)
		mapping.Type(typeName)

		json := map[string]interface{}{typeName: properties}
		mapping.BodyJson(json)

		rsp, err := mapping.Do(ctx)
		if err != nil {
			prod.Log.Error.Println("Issue during creating index mapping: ", err)
		}
		fmt.Println("map result: ", rsp)
	}

	return err
}

func (prod *ElasticSearch) sendMessage(msg *core.Message) {
	client, err := prod.getClient()
	if err != nil {
		prod.Log.Error.Print("Sending messages failed: ", err)
		return
	}

	ctx := context.Background()

	index := prod.indexMap[msg.GetStreamID()]
	typeName := prod.typeMap[msg.GetStreamID()]

	fmt.Println("Index: ", index)
	fmt.Println("typeName: ", typeName)

	put, err := client.
		Index().
		Index(index).
		Type(typeName).
		BodyString(msg.String()).
		Do(ctx)
	if err != nil {
		prod.Log.Error.Println("Can not add document to elastic: ", err)
		return
	}
	fmt.Printf("Indexed tweet %s to index %s, type %s\n", put.Id, put.Index, put.Type)
}
