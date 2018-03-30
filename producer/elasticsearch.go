// Copyright 2015-2018 trivago N.V.
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
	"net/http"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo"
	"github.com/trivago/tgo/tcontainer"
	"gopkg.in/olivere/elastic.v5"
)

// ElasticSearch producer plugin
//
// The ElasticSearch producer sends messages to elastic search using the bulk
// http API. The producer expects a json payload.
//
// Parameters
//
// - Retry/Count: Set the amount of retries before a Elasticsearch request
// fail finally.
// By default this parameter is set to "3".
//
// - Retry/TimeToWaitSec: This value denotes the time in seconds after which a
// failed dataset will be  transmitted again.
// By default this parameter is set to "3".
//
// - SetGzip: This value enables or disables gzip compression for Elasticsearch
// requests (disabled by default). This option is used one to one for the library
// package. See http://godoc.org/gopkg.in/olivere/elastic.v5#SetGzip
// By default this parameter is set to "false".
//
// - Servers: This value defines a list of servers to connect to.
//
// - User: This value used as the username for the elasticsearch server.
// By default this parameter is set to "".
//
// - Password: This value used as the password for the elasticsearch server.
// By default this parameter is set to "".
//
// - StreamProperties: This value defines the mapping and settings for each stream.
// As index use the stream name here.
//
// - StreamProperties/<streamName>/Index: The value defines the Elasticsearch
// index used for the stream.
//
// - StreamProperties/<streamName>/Type: This value defines the document type
// used for the stream.
//
// - StreamProperties/<streamName>/TimeBasedIndex: This value can be set to "true"
// to append the date of the message to the index as in "<index>_<TimeBasedFormat>".
// NOTE: This setting incurs a performance penalty because it is necessary to
// check if an index exists for each message!
// By default this parameter is set to "false".
//
// - StreamProperties/<streamName>/TimeBasedFormat: This value can be set to a valid
// go time format string to be used with DayBasedIndex.
// By default this parameter is set to "2006-01-02".
//
// - StreamProperties/<streamName>/Mapping: This value is a map which is used
// for the document field mapping. As document type, the already defined type is
// reused for the field mapping. See
// https://www.elastic.co/guide/en/elasticsearch/reference/5.4/indices-create-index.html#mappings
//
// - StreamProperties/<streamName>/Settings: This value is a map which is used
// for the index settings. See
// https://www.elastic.co/guide/en/elasticsearch/reference/5.4/indices-create-index.html#mappings
//
// Examples
//
// This example starts a simple twitter example producer for local running ElasticSearch:
//
//  producerElasticSearch:
//    Type: producer.ElasticSearch
//    Streams: tweets_stream
//    SetGzip: true
//    Servers:
//      - http://127.0.0.1:9200
//    StreamProperties:
//      tweets_stream:
//        Index: twitter
//        DayBasedIndex: true
//        Type: tweet
//        Mapping:
//          # index mapping for payload
//          user: keyword
//          message: text
//        Settings:
//          number_of_shards: 1
//          number_of_replicas: 1
type ElasticSearch struct {
	core.BatchedProducer `gollumdoc:"embed_type"`
	connection           elasticConnection
	indexMap             map[core.MessageStreamID]*indexMapItem
}

type indexMapItem struct {
	name         string
	typeName     string
	settings     *elasticIndex
	useTimeIndex bool
	timeFormat   string
}

func newIndexMapItem() *indexMapItem {
	return &indexMapItem{
		timeFormat: "2006-01-02",
	}
}

func (item *indexMapItem) GetIndexName(t time.Time) string {
	if item.useTimeIndex {
		return item.name + t.Format(item.timeFormat)
	}
	return item.name
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

func newElasticIndex(property tcontainer.MarshalMap) *elasticIndex {
	elType, _ := property.String("Type")
	mapping, _ := property.MarshalMap("Mapping")
	settings, _ := property.MarshalMap("Settings")

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
	prod.connection.retrier.logger = prod.Logger.WithField("Scope", "connection.retrier")

	prod.Logger.Debugf("Using retrier with a retry count of '%d' and a time to wait of '%d' sec", retry, timeToWaitSec)
}

func (prod *ElasticSearch) configureIndexSettings(properties tcontainer.MarshalMap, errors *tgo.ErrorStack) {
	prod.indexMap = map[core.MessageStreamID]*indexMapItem{}

	if len(properties) <= 0 {
		prod.Logger.Error("No stream configuration found. Please check your config.")
		return
	}

	for streamName := range properties {
		streamID := core.GetStreamID(streamName)
		indexMapItem := newIndexMapItem()

		property, err := properties.MarshalMap(streamName)
		if err != nil {
			prod.Logger.Errorf("no configuration found for stream '%s'. Please check your config.", streamName)
			errors.Push(err)
			continue
		}

		indexMapItem.name, err = property.String("Index")
		if err != nil {
			prod.Logger.Errorf("no index configured for stream '%s'. Please check your config.", streamName)
			errors.Push(err)
			continue
		}

		indexMapItem.useTimeIndex, _ = property.Bool("TimeBasedIndex")
		timeFormat, _ := property.String("TimeBasedFormat")
		if len(timeFormat) == 0 {
			timeFormat = "2006-01-02"
		}
		indexMapItem.timeFormat = "_" + timeFormat

		indexMapItem.typeName, err = property.String("Type")
		if err != nil {
			prod.Logger.Errorf("no data type configured for stream '%s'. Please check your config.", streamName)
		}

		indexMapItem.settings = newElasticIndex(property)
		prod.indexMap[streamID] = indexMapItem
	}
}

func (prod *ElasticSearch) getClient() *elastic.Client {
	if prod.connection.isConnected() {
		return prod.connection.client
	}

	if err := prod.connection.connect(); err != nil {
		prod.Logger.WithError(err).Error("Error during connection")
		return nil
	}

	return prod.connection.client
}

func (prod *ElasticSearch) indexExists(client *elastic.Client, indexName string) bool {
	exists, err := client.IndexExists(indexName).Do(context.Background())
	if err != nil {
		prod.Logger.WithError(err).Error("Error during index check")
		return false
	}
	return exists
}

func (prod *ElasticSearch) createIndexIfRequired(indexName string, settings *elasticIndex) bool {
	client := prod.getClient()
	if client == nil {
		return false
	}

	if !prod.indexExists(client, indexName) {
		if _, err := client.CreateIndex(indexName).Do(context.Background()); err != nil {
			prod.Logger.WithError(err).Errorln("Failed to create index")
			return false
		}
		prod.Logger.Debugf("Created index %s", indexName)
	}

	if settings == nil {
		prod.Logger.Debugf("No settings for index %s", indexName)
		return true
	}

	for typeName, properties := range settings.Mappings {
		mapping := client.PutMapping()
		mapping.Index(indexName)
		mapping.Type(typeName)

		json := map[string]interface{}{typeName: properties}
		mapping.BodyJson(json)

		_, err := mapping.Do(context.Background())
		if err != nil {
			prod.Logger.WithError(err).Errorf("Error creating mapping for type %s.%s", indexName, typeName)
		}
	}

	return true
}

func (prod *ElasticSearch) submitMessages(messages []*core.Message) {
	client := prod.getClient()
	if client == nil {
		prod.Logger.Error("Failed to get client. Cannot send messages")
	}

	// Handle time based index creation
	timeBasedIndexes := make(map[string]*elasticIndex)
	for _, msg := range messages {
		if item, isSet := prod.indexMap[msg.GetStreamID()]; isSet && item.useTimeIndex {
			timeBasedIndexes[item.GetIndexName(msg.GetCreationTime())] = item.settings
		}
	}

	for indexName, settings := range timeBasedIndexes {
		prod.createIndexIfRequired(indexName, settings)
	}

	// Send messages
	bulkRequest := client.Bulk()
	for _, msg := range messages {
		indexMapItem, isSet := prod.indexMap[msg.GetStreamID()]
		if !isSet {
			prod.Logger.Warningf("No index setting for stream %s", msg.GetStreamID().GetName())
			continue
		}

		bulkIndexRequest := elastic.NewBulkIndexRequest()
		bulkIndexRequest.Index(indexMapItem.GetIndexName(msg.GetCreationTime())).
			Type(indexMapItem.typeName).
			Doc(msg.String())

		bulkRequest.Add(bulkIndexRequest)
	}

	// NumberOfActions contains the number of requests in a bulk
	prod.Logger.Debugf("bulkRequest.NumberOfActions: %d", bulkRequest.NumberOfActions())

	// Do sends the bulk requests to Elasticsearch
	bulkResponse, err := bulkRequest.Do(context.Background())
	if err != nil {
		prod.Logger.Error(err)
	}

	// Bulk request actions get cleared
	numberOfActionsAfter := bulkRequest.NumberOfActions()
	if numberOfActionsAfter != 0 {
		prod.Logger.Errorf("Could not send '%d' messages to Elasticsearch", numberOfActionsAfter)
	}

	if bulkResponse != nil {
		// Indexed returns information abount indexed documents
		indexed := bulkResponse.Indexed()
		prod.Logger.Debugf("%d messages indexed successfully in Elasticsearch", len(indexed))

		// Created returns information about created documents
		created := bulkResponse.Created()
		prod.Logger.Debugf("%d messages created successfully in Elasticsearch", len(created))
	}
}

// Produce starts the producer
func (prod *ElasticSearch) Produce(workers *sync.WaitGroup) {
	defer prod.WorkerDone()

	// create all indexes that are not time based
	for _, item := range prod.indexMap {
		if !item.useTimeIndex {
			prod.createIndexIfRequired(item.name, item.settings)
		}
	}

	prod.AddMainWorker(workers)
	prod.BatchMessageLoop(workers, func() core.AssemblyFunc { return prod.submitMessages })
}

// -- elasticConnection --

type elasticConnection struct {
	retrier           retrier
	client            *elastic.Client
	servers           []string
	user              string
	password          string
	setGzip           bool
	isConnectedStatus bool
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
	logger  logrus.FieldLogger
	backoff elastic.Backoff
	retry   int
}

// Retry implements type Retrier interface
// see: https://github.com/olivere/elastic/wiki/Retrier-and-Backoff
func (r *retrier) Retry(ctx context.Context, retry int, req *http.Request, resp *http.Response, err error) (time.Duration, bool, error) {
	// Fail hard on a specific error
	if err == syscall.ECONNREFUSED {
		err = errors.New("Elasticsearch or network down")
		r.logger.Error(err)
		return 0, false, err
	}

	// Stop after n retries
	if retry >= r.retry {
		r.logger.Debugf("Stop retrying after '%d' retries", retry)
		return 0, false, nil
	}

	// Let the backoff strategy decide how long to wait and whether to stop
	r.logger.Debugln("Retry to connect to Elasticsearch")
	wait, stop := r.backoff.Next(retry)
	return wait, stop, nil
}
