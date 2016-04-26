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
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tnet"
	"gopkg.in/redis.v3"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Redis producer plugin
// This producer sends data to a redis server. Different redis storage types
// and database indexes are supported. This producer does not implement support
// for redis 3.0 cluster.
// Configuration example
//
//  - "producer.Redis":
//    Address: ":6379"
//    Database: 0
//    Key: "default"
//    Storage: "hash"
//    FieldFormat: "format.Identifier"
//    FieldAfterFormat: false
//
// Address stores the identifier to connect to.
// This can either be any ip address and port like "localhost:6379" or a file
// like "unix:///var/redis.socket". By default this is set to ":6379".
// This producer does not implement a fuse breaker.
//
// Database defines the redis database to connect to.
// By default this is set to 0.
//
// Key defines the redis key to store the values in.
// By default this is set to "default".
//
// Storage defines the type of the storage to use. Valid values are: "hash",
// "list", "set", "sortedset", "string". By default this is set to "hash".
//
// FieldFormat defines an extra formatter used to define an additional field or
// score value if required by the storage type. If no field value is required
// this value is ignored. By default this is set to "format.Identifier".
//
// FieldAfterFormat will send the formatted message to the FieldFormatter if set
// to true. If this is set to false the message will be send to the FieldFormatter
// before it has been formatted. By default this is set to false.
type Redis struct {
	core.BufferedProducer
	address         string
	protocol        string
	password        string
	database        int64
	key             string
	client          *redis.Client
	store           func(*core.Message)
	fieldFormatters []core.Formatter
	fieldFromParsed bool
}

func init() {
	core.TypeRegistry.Register(Redis{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *Redis) Configure(conf core.PluginConfigReader) error {
	prod.BufferedProducer.Configure(conf)

	plugins, err := conf.WithError.GetPluginArray("FieldFormatters", []core.Plugin{})
	if !conf.Errors.Push(err) {
		for _, plugin := range plugins {
			formatter, isFormatter := plugin.(core.Formatter)
			if !isFormatter {
				conf.Errors.Pushf("Plugin is not a valid formatter")
			}
			prod.fieldFormatters = append(prod.fieldFormatters, formatter)
		}
	}

	prod.SetStopCallback(prod.close)

	prod.password = conf.GetString("Password", "")
	prod.database = int64(conf.GetInt("Database", 0))
	prod.key = conf.GetString("Key", "default")
	prod.fieldFromParsed = conf.GetBool("FieldAfterFormat", false)
	prod.address, prod.protocol = tnet.ParseAddress(conf.GetString("Address", ":6379"))

	switch strings.ToLower(conf.GetString("Storage", "hash")) {
	case "hash":
		prod.store = prod.storeHash
	case "list":
		prod.store = prod.storeList
	case "set":
		prod.store = prod.storeSet
	case "sortedset":
		prod.store = prod.storeSortedSet
	default:
		fallthrough
	case "string":
		prod.store = prod.storeString
	}

	return conf.Errors.OrNil()
}

func (prod *Redis) formatField(msg *core.Message) []byte {
	fieldMsg := *msg
	for _, formatter := range prod.fieldFormatters {
		formatter.Format(&fieldMsg)
	}
	return fieldMsg.Data
}

func (prod *Redis) storeHash(msg *core.Message) {
	originalMsg := *msg
	prod.Format(msg)

	var field []byte
	if prod.fieldFromParsed {
		field = prod.formatField(msg)
	} else {
		field = prod.formatField(&originalMsg)
	}

	result := prod.client.HSet(prod.key, string(field), string(msg.Data))
	if result.Err() != nil {
		prod.Log.Error.Print("Redis: ", result.Err())
		prod.Drop(&originalMsg)
	}
}

func (prod *Redis) storeList(msg *core.Message) {
	originalMsg := *msg
	prod.Format(msg)

	result := prod.client.RPush(prod.key, string(msg.Data))
	if result.Err() != nil {
		prod.Log.Error.Print("Redis: ", result.Err())
		prod.Drop(&originalMsg)
	}
}

func (prod *Redis) storeSet(msg *core.Message) {
	originalMsg := *msg
	prod.Format(msg)

	result := prod.client.SAdd(prod.key, string(msg.Data))
	if result.Err() != nil {
		prod.Log.Error.Print("Redis: ", result.Err())
		prod.Drop(&originalMsg)
	}
}

func (prod *Redis) storeSortedSet(msg *core.Message) {
	originalMsg := *msg
	prod.Format(msg)

	var scoreValue []byte
	if prod.fieldFromParsed {
		scoreValue = prod.formatField(msg)
	} else {
		scoreValue = prod.formatField(&originalMsg)
	}

	score, err := strconv.ParseFloat(string(scoreValue), 64)
	if err != nil {
		prod.Log.Error.Print("Redis: ", err)
		return // ### return, no valid score ###
	}

	result := prod.client.ZAdd(prod.key, redis.Z{
		Score:  score,
		Member: string(msg.Data),
	})

	if result.Err() != nil {
		prod.Log.Error.Print("Redis: ", result.Err())
		prod.Drop(&originalMsg)
	}
}

func (prod *Redis) storeString(msg *core.Message) {
	originalMsg := *msg
	prod.Format(msg)

	result := prod.client.Set(prod.key, string(msg.Data), time.Duration(0))
	if result.Err() != nil {
		prod.Log.Error.Print("Redis: ", result.Err())
		prod.Drop(&originalMsg)
	}
}

func (prod *Redis) close() {
	defer prod.WorkerDone()
	prod.DefaultClose()
}

// Produce writes to stdout or stderr.
func (prod *Redis) Produce(workers *sync.WaitGroup) {
	prod.client = redis.NewClient(&redis.Options{
		Addr:     prod.address,
		Network:  prod.protocol,
		Password: prod.password,
		DB:       prod.database,
	})

	if _, err := prod.client.Ping().Result(); err != nil {
		prod.Log.Error.Print("Redis: ", err)
	}

	prod.AddMainWorker(workers)
	prod.MessageControlLoop(prod.store)
}
