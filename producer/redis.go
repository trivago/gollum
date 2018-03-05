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
	"github.com/go-redis/redis"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tnet"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Redis producer
//
// This producer sends messages to a redis server. Different redis storage types
// and database indexes are supported. This producer does not implement support
// for redis 3.0 cluster.
//
// Parameters
//
// - Address: Stores the identifier to connect to.
// This can either be any ip address and port like "localhost:6379" or a file
// like "unix:///var/redis.socket". By default this is set to ":6379".
//
// - Database: Defines the redis database to connect to.
//
// - Key: Defines the redis key to store the values in.
// This field is ignored when "KeyFormatter" is set.
// By default this is set to "default".
//
// - Storage: Defines the type of the storage to use. Valid values are: "hash",
// "list", "set", "sortedset", "string". By default this is set to "hash".
//
// - KeyFrom: Defines the name of the metadata field used as a key for messages
// sent to redis. If the name is an empty string no key is sent. By default
// this value is set to an empty string.
//
// - FieldFrom: Defines the name of the metadata field used as a field for messages
// sent to redis. If the name is an empty string no key is sent. By default
// this value is set to an empty string.
//
// Examples
//
// .
//
//   RedisProducer00:
//     Type: producer.Redis
//     Address: ":6379"
//     Key: "mykey"
//     Storage: "hash"
//
type Redis struct {
	core.BufferedProducer `gollumdoc:"embed_type"`
	address               string
	protocol              string
	password              string `config:"Password"`
	database              int    `config:"Database" default:"0"`
	key                   string `config:"KeyFrom"`
	field                 string `config:"FieldFrom"`
	client                *redis.Client
	store                 func(msg *core.Message)
}

func init() {
	core.TypeRegistry.Register(Redis{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *Redis) Configure(conf core.PluginConfigReader) {
	prod.SetStopCallback(prod.close)

	prod.protocol, prod.address = tnet.ParseAddress(conf.GetString("Address", ":6379"), "tcp")

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
}

func (prod *Redis) getValueAndKey(msg *core.Message) (v, k []byte) {
	meta := msg.GetMetadata()
	key := meta.GetValue(prod.key)

	return msg.GetPayload(), key
}

func (prod *Redis) getValueFieldAndKey(msg *core.Message) (v, f, k []byte) {
	meta := msg.GetMetadata()
	key := meta.GetValue(prod.key)
	field := meta.GetValue(prod.field)

	return msg.GetPayload(), field, key
}

func (prod *Redis) storeHash(msg *core.Message) {
	value, field, key := prod.getValueFieldAndKey(msg)
	result := prod.client.HSet(string(key), string(field), string(value))
	if result.Err() != nil {
		prod.Logger.Error("Redis: ", result.Err())
		prod.TryFallback(msg)
	}
}

func (prod *Redis) storeList(msg *core.Message) {
	value, key := prod.getValueAndKey(msg)

	result := prod.client.RPush(string(key), string(value))
	if result.Err() != nil {
		prod.Logger.Error("Redis: ", result.Err())
		prod.TryFallback(msg)
	}
}

func (prod *Redis) storeSet(msg *core.Message) {
	value, key := prod.getValueAndKey(msg)

	result := prod.client.SAdd(string(key), string(value))
	if result.Err() != nil {
		prod.Logger.Error("Redis: ", result.Err())
		prod.TryFallback(msg)
	}
}

func (prod *Redis) storeSortedSet(msg *core.Message) {
	value, scoreValue, key := prod.getValueFieldAndKey(msg)
	score, err := strconv.ParseFloat(string(scoreValue), 64)
	if err != nil {
		prod.Logger.Error("Redis: ", err)
		return // ### return, no valid score ###
	}

	result := prod.client.ZAdd(string(key),
		redis.Z{
			Score:  score,
			Member: string(value),
		})

	if result.Err() != nil {
		prod.Logger.Error("Redis: ", result.Err())
		prod.TryFallback(msg)
	}
}

func (prod *Redis) storeString(msg *core.Message) {
	value, key := prod.getValueAndKey(msg)

	result := prod.client.Set(string(key), string(value), time.Duration(0))
	if result.Err() != nil {
		prod.Logger.Error("Redis: ", result.Err())
		prod.TryFallback(msg)
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
		prod.Logger.Error("Redis: ", err)
	}

	prod.AddMainWorker(workers)
	prod.MessageControlLoop(prod.store)
}
