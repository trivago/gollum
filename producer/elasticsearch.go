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
	"github.com/pkg/errors"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo"
	"github.com/trivago/tgo/tcontainer"
	"github.com/trivago/tgo/tlog"
	"gopkg.in/olivere/elastic.v5"
	"net/http"
	"sync"
	"syscall"
	"time"
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
//    #Connections: 6
//    Retry:
//		Count: 3
//		TimeToWaitSec: 3
//	  SetGzip: true
//
//    User: ""
//    Password: ""
//    Servers:
//      - http://127.0.0.1:9200
//    StreamProperties:
//		<streamName>:
//			Index: twitter
// 			DayBasedIndex: false
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
// SetGzip enables or disables gzip compression (disabled by default).
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
type ElasticSearch struct {
	core.BatchedProducer `gollumdoc:"embed_type"`
	connection           elasticConnection
	indexMap             map[core.MessageStreamID]*indexMapItem
}

type indexMapItem struct {
	name        string
	typeName    string
	settings    *elasticIndex
	useDayIndex bool
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

type indexExists uint8

const (
	indexExistsFalse indexExists = iota
	indexExistsTrue
	indexExistsAuto
)

func init() {
	core.TypeRegistry.Register(ElasticSearch{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *ElasticSearch) Configure(conf core.PluginConfigReader) {
	prod.connection.servers = conf.GetStringArray("Servers", []string{"http://127.0.0.1:9200"})
	prod.connection.user = conf.GetString("User", "")
	prod.connection.password = conf.GetString("Password", "")
	prod.connection.setGzip = conf.GetBool("SetGzip", false)
	prod.connection.isConnectedStatus = false

	prod.configureIndexSettings(conf.GetMap("StreamProperties", tcontainer.NewMarshalMap()), conf.Errors)
	prod.configureRetrySettings(conf.GetInt("Retry/Count", 3), conf.GetInt("Retry/TimeToWaitSec", 3))
}

func (prod *ElasticSearch) configureRetrySettings(retry, timeToWaitSec int64) {
	prod.connection.retrier.retry = int(retry)
	prod.connection.retrier.backoff = elastic.NewConstantBackoff(time.Duration(timeToWaitSec) * time.Second)
	prod.connection.retrier.log = &prod.Log

	prod.Log.Debug.Printf("Using retrier with a retry count of '%d' and a time to wait of '%d' sec", retry, timeToWaitSec)
}

func (prod *ElasticSearch) newIndexMapItem() *indexMapItem {
	return &indexMapItem{
		"",
		"",
		nil,
		false,
	}
}

func (prod *ElasticSearch) configureIndexSettings(properties tcontainer.MarshalMap, errors *tgo.ErrorStack) {
	prod.indexMap = map[core.MessageStreamID]*indexMapItem{}

	if len(properties) <= 0 {
		prod.Log.Error.Print("No stream configuration found. Please check your config.")
		return
	}

	for streamName := range properties {
		streamID := core.GetStreamID(streamName)
		indexMapItem := prod.newIndexMapItem()

		property, err := properties.MarshalMap(streamName)
		if err != nil {
			prod.Log.Error.Printf("no configuration found for stream '%s'. Please check your config.", streamName)
			errors.Push(err)
			continue
		}

		indexName, err := property.String("index")
		if err != nil {
			prod.Log.Error.Printf("no index configured for stream '%s'. Please check your config.", streamName)
			errors.Push(err)
			continue
		}

		dayBasedIndex, _ := property.Bool("daybasedindex")
		if dayBasedIndex {
			currentTime := time.Now()
			indexName += currentTime.Format("_2006-01-02")
			indexMapItem.useDayIndex = true
		}

		typeName, err := property.String("type")
		if err != nil {
			prod.Log.Error.Printf("no data type configured for stream '%s'. Please check your config.", streamName)
		}

		indexMapItem.settings = prod.newElasticIndex(property)
		indexMapItem.typeName = typeName
		indexMapItem.name = indexName

		prod.indexMap[streamID] = indexMapItem
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
	prod.BatchMessageLoop(workers, prod.submitMessages)
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
	for _, indexMapItem := range prod.indexMap {
		prod.createIndex(indexMapItem.name, indexMapItem.settings, indexExistsAuto)
	}
}

func (prod *ElasticSearch) createIndex(indexName string, elIndex *elasticIndex, indexExistsCheck indexExists) error {
	client, err := prod.getClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// avoid unnecessary request for checking index (see submitMessage())
	var exists bool
	switch indexExistsCheck {
	case indexExistsTrue:
		exists = true
		break
	case indexExistsFalse:
		exists = false
		break
	case indexExistsAuto:
		exists, err = prod.isIndexExists(indexName, client, ctx)
		if err != nil {
			return err
		}
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

		_, err := mapping.Do(ctx)
		if err != nil {
			prod.Log.Error.Println("Issue during creating index mapping: ", err)
		}
	}

	return err
}

func (prod *ElasticSearch) isIndexExists(indexName string, client *elastic.Client, ctx context.Context) (bool, error) {
	exists, err := client.IndexExists(indexName).Do(ctx)
	prod.Log.Debug.Printf("Index %s exists: %v", indexName, exists)
	if err != nil {
		prod.Log.Error.Println("Issue during checking index: ", err)
		return false, err
	}

	return exists, nil
}

func (prod *ElasticSearch) submitMessages() core.AssemblyFunc {
	return func(messages []*core.Message) {
		client, err := prod.getClient()
		if err != nil {
			prod.Log.Error.Print("Sending messages failed: ", err)
			return
		}

		bulkRequest := client.Bulk()
		ctx := context.Background()

		for _, msg := range messages {
			indexMapItem := prod.indexMap[msg.GetStreamID()]

			if indexMapItem.useDayIndex {
				// create daily index ...
				indexExists, _ := prod.isIndexExists(indexMapItem.name, client, ctx)
				if !indexExists {
					prod.createIndex(indexMapItem.name, indexMapItem.settings, indexExistsFalse)
				}
			}

			bulkIndexRequest := elastic.NewBulkIndexRequest()
			bulkIndexRequest.Index(indexMapItem.name).
				Type(indexMapItem.typeName).
				Doc(msg.String())

			bulkRequest.Add(bulkIndexRequest)
		}

		// NumberOfActions contains the number of requests in a bulk
		prod.Log.Debug.Printf("bulkRequest.NumberOfActions: %d", bulkRequest.NumberOfActions())

		// Do sends the bulk requests to Elasticsearch
		bulkResponse, err := bulkRequest.Do(ctx)
		if err != nil {
			prod.Log.Error.Print(err)
		}

		// Bulk request actions get cleared
		numberOfActionsAfter := bulkRequest.NumberOfActions()
		if numberOfActionsAfter != 0 {
			prod.Log.Error.Printf("Could not send '%d' messages to Elasticsearch", numberOfActionsAfter)
		}

		if bulkResponse != nil {
			// Indexed returns information abount indexed documents
			indexed := bulkResponse.Indexed()
			prod.Log.Debug.Printf("%d messages indexed successfully in Elasticsearch", len(indexed))

			// Created returns information about created documents
			created := bulkResponse.Created()
			prod.Log.Debug.Printf("%d messages created successfully in Elasticsearch", len(created))
		}
	}
}

// -- elasticConnection --

type elasticConnection struct {
	servers           []string
	user              string
	password          string
	setGzip           bool
	client            *elastic.Client
	isConnectedStatus bool
	retrier           retrier
}

func (conn *elasticConnection) isConnected() bool {
	return conn.isConnectedStatus
}

func (conn *elasticConnection) connect() error {
	conf := []elastic.ClientOptionFunc{elastic.SetURL(conn.servers...), elastic.SetSniff(false), elastic.SetGzip(conn.setGzip)}
	if len(conn.user) > 0 {
		conf = append(conf, elastic.SetBasicAuth(conn.user, conn.password))
	}

	if conn.retrier.retry > 0 {
		conf = append(conf, elastic.SetRetrier(&conn.retrier))
	}

	client, err := elastic.NewClient(conf...)
	if err != nil {
		return err
	}

	conn.client = client
	conn.isConnectedStatus = true

	return nil
}

// -- retrier --

type retrier struct {
	retry   int
	backoff elastic.Backoff
	log     *tlog.LogScope
}

// Retry implements type Retrier interface
// see: https://github.com/olivere/elastic/wiki/Retrier-and-Backoff
func (r *retrier) Retry(ctx context.Context, retry int, req *http.Request, resp *http.Response, err error) (time.Duration, bool, error) {
	// Fail hard on a specific error
	if err == syscall.ECONNREFUSED {
		err = errors.New("Elasticsearch or network down")
		r.log.Error.Print(err)
		return 0, false, err
	}

	// Stop after n retries
	if retry >= r.retry {
		r.log.Debug.Printf("Stop retrying after '%d' retries", retry)
		return 0, false, nil
	}

	// Let the backoff strategy decide how long to wait and whether to stop
	r.log.Debug.Println("Retry to connect to Elasticsearch")
	wait, stop := r.backoff.Next(retry)
	return wait, stop, nil
}
