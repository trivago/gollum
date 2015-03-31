// Copyright 2015 trivago GmbH
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
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	"gopkg.in/redis.v2"
	"strconv"
	"strings"
	"sync"
)

// Redis producer plugin
// Configuration example
//
//   - "zup.Redis":
//     Enable: true
//     Address: "127.0.0.1:6379"
//	   Database: 0
//	   Key: "gollum"
//     Storage: "hash"
//
// Address defines the redis server to connect to.
// This must be a ip address and port like "127.0.0.1:6379"
// By default this is set to "127.0.0.1:6379".
type Redis struct {
	core.ProducerBase
	address   string
	protocol  string
	password  string
	database  int64
	key       string
	client    *redis.Client
	store     func(msg core.Message)
	keyFormat core.Formatter
}

func init() {
	shared.RuntimeType.Register(Redis{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *Redis) Configure(conf core.PluginConfig) error {
	err := prod.ProducerBase.Configure(conf)
	if err != nil {
		return err
	}

	plugin, err := core.NewPluginWithType(conf.GetString("KeyFormatter", "format.Identifier"), conf)
	if err != nil {
		return err // ### return, plugin load error ###
	}

	prod.keyFormat = plugin.(core.Formatter)
	prod.password = conf.GetString("Password", "")
	prod.database = int64(conf.GetInt("Database", 0))
	prod.key = conf.GetString("Key", "default")
	prod.address, prod.protocol = shared.ParseAddress(conf.GetString("Address", ":6379"))

	switch strings.ToLower(conf.GetString("Storage", "string")) {
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

	return nil
}

func (prod *Redis) storeHash(msg core.Message) {
	prod.keyFormat.PrepareMessage(msg)
	prod.Formatter().PrepareMessage(msg)

	key := prod.keyFormat.String()
	value := prod.Formatter().String()

	result := prod.client.HSet(prod.key, key, value)
	if result.Err() != nil {
		Log.Error.Print("Redis: ", result.Err())
		msg.Drop(prod.GetTimeout())
	}
}

func (prod *Redis) storeList(msg core.Message) {
	prod.Formatter().PrepareMessage(msg)
	value := prod.Formatter().String()

	result := prod.client.RPush(prod.key, value)
	if result.Err() != nil {
		Log.Error.Print("Redis: ", result.Err())
		msg.Drop(prod.GetTimeout())
	}
}

func (prod *Redis) storeSet(msg core.Message) {
	prod.Formatter().PrepareMessage(msg)
	value := prod.Formatter().String()

	result := prod.client.SAdd(prod.key, value)
	if result.Err() != nil {
		Log.Error.Print("Redis: ", result.Err())
		msg.Drop(prod.GetTimeout())
	}
}

func (prod *Redis) storeSortedSet(msg core.Message) {
	prod.keyFormat.PrepareMessage(msg)
	prod.Formatter().PrepareMessage(msg)

	scoreVal, err := strconv.ParseFloat(prod.keyFormat.String(), 64)
	if err != nil {
		Log.Error.Print("Redis: ", err)
		return // ### return, no valid score ###
	}

	result := prod.client.ZAdd(prod.key, redis.Z{
		Score:  scoreVal,
		Member: prod.Formatter().String(),
	})

	if result.Err() != nil {
		Log.Error.Print("Redis: ", result.Err())
		msg.Drop(prod.GetTimeout())
	}
}

func (prod *Redis) storeString(msg core.Message) {
	prod.Formatter().PrepareMessage(msg)
	value := prod.Formatter().String()

	result := prod.client.Set(prod.key, value)
	if result.Err() != nil {
		Log.Error.Print("Redis: ", result.Err())
		msg.Drop(prod.GetTimeout())
	}
}

// Produce writes to stdout or stderr.
func (prod Redis) Produce(workers *sync.WaitGroup) {
	prod.client = redis.NewClient(&redis.Options{
		Addr:     prod.address,
		Network:  prod.protocol,
		Password: prod.password,
		DB:       prod.database,
	})

	if _, err := prod.client.Ping().Result(); err != nil {
		Log.Error.Print("Redis: ", err)
	}

	prod.AddMainWorker(workers)
	defer prod.WorkerDone()

	prod.DefaultControlLoop(prod.store, nil)
}
