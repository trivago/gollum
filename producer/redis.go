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
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	"gopkg.in/redis.v4"
	"strconv"
	"strings"
	"sync"
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
//    FieldFormatter: "format.Identifier"
//    FieldAfterFormat: false
//    KeyFormatter: "format.Forward"
//    KeyAfterFormat: false
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
// This field is ignored when "KeyFormatter" is set.
// By default this is set to "default".
//
// Storage defines the type of the storage to use. Valid values are: "hash",
// "list", "set", "sortedset", "string". By default this is set to "hash".
//
// FieldFormatter defines an extra formatter used to define an additional field or
// score value if required by the storage type. If no field value is required
// this value is ignored. By default this is set to "format.Identifier".
//
// FieldAfterFormat will send the formatted message to the FieldFormatter if set
// to true. If this is set to false the message will be send to the FieldFormatter
// before it has been formatted. By default this is set to false.
//
// KeyFormatter defines an extra formatter used to allow generating the key from
// a message. If this value is set the "Key" field will be ignored. By default
// this field is not used.
//
// KeyAfterFormat will send the formatted message to the keyFormatter if set
// to true. If this is set to false the message will be send to the keyFormatter
// before it has been formatted. By default this is set to false.
type Redis struct {
	core.ProducerBase
	address         string
	protocol        string
	password        string
	database        int
	key             string
	client          *redis.Client
	store           func(msg core.Message)
	fieldFormat     core.Formatter
	keyFormat       core.Formatter
	fieldFromParsed bool
	keyFromParsed   bool
}

func init() {
	shared.TypeRegistry.Register(Redis{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *Redis) Configure(conf core.PluginConfig) error {
	err := prod.ProducerBase.Configure(conf)
	if err != nil {
		return err
	}
	prod.SetStopCallback(prod.close)

	fieldFormat, err := core.NewPluginWithType(conf.GetString("FieldFormatter", "format.Identifier"), conf)
	if err != nil {
		return err // ### return, plugin load error ###
	}
	prod.fieldFormat = fieldFormat.(core.Formatter)

	if conf.HasValue("KeyFormatter") {
		keyFormat, err := core.NewPluginWithType(conf.GetString("KeyFormatter", "format.Forward"), conf)
		if err != nil {
			return err // ### return, plugin load error ###
		}
		prod.keyFormat = keyFormat.(core.Formatter)
	}

	prod.password = conf.GetString("Password", "")
	prod.database = conf.GetInt("Database", 0)
	prod.key = conf.GetString("Key", "default")
	prod.fieldFromParsed = conf.GetBool("FieldAfterFormat", false)
	prod.keyFromParsed = conf.GetBool("KeyAfterFormat", false)
	prod.address, prod.protocol = shared.ParseAddress(conf.GetString("Address", ":6379"))

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

	return nil
}

func (prod *Redis) getValueAndKey(msg core.Message) (v []byte, k string) {
	value, _ := prod.Format(msg)

	if prod.keyFormat == nil {
		return value, prod.key
	}

	if prod.keyFromParsed {
		keyMsg := msg
		keyMsg.Data = value
		key, _ := prod.keyFormat.Format(keyMsg)
		return value, string(key)
	}

	key, _ := prod.keyFormat.Format(msg)
	return value, string(key)
}

func (prod *Redis) getValueFieldAndKey(msg core.Message) (v []byte, f []byte, k string) {
	value, _ := prod.Format(msg)
	key := []byte(prod.key)

	if prod.keyFormat != nil {
		if prod.keyFromParsed {
			keyMsg := msg
			keyMsg.Data = value
			key, _ = prod.keyFormat.Format(keyMsg)
		} else {
			key, _ = prod.keyFormat.Format(msg)
		}
	}

	if prod.fieldFromParsed {
		fieldMsg := msg
		fieldMsg.Data = value
		field, _ := prod.fieldFormat.Format(fieldMsg)
		return value, field, string(key)
	}

	field, _ := prod.fieldFormat.Format(msg)
	return value, field, string(key)
}

func (prod *Redis) storeHash(msg core.Message) {
	value, field, key := prod.getValueFieldAndKey(msg)
	result := prod.client.HSet(key, string(field), string(value))
	if result.Err() != nil {
		Log.Error.Print("Redis: ", result.Err())
		prod.Drop(msg)
	}
}

func (prod *Redis) storeList(msg core.Message) {
	value, key := prod.getValueAndKey(msg)

	result := prod.client.RPush(key, string(value))
	if result.Err() != nil {
		Log.Error.Print("Redis: ", result.Err())
		prod.Drop(msg)
	}
}

func (prod *Redis) storeSet(msg core.Message) {
	value, key := prod.getValueAndKey(msg)

	result := prod.client.SAdd(key, string(value))
	if result.Err() != nil {
		Log.Error.Print("Redis: ", result.Err())
		prod.Drop(msg)
	}
}

func (prod *Redis) storeSortedSet(msg core.Message) {
	value, scoreValue, key := prod.getValueFieldAndKey(msg)
	score, err := strconv.ParseFloat(string(scoreValue), 64)
	if err != nil {
		Log.Error.Print("Redis: ", err)
		return // ### return, no valid score ###
	}

	result := prod.client.ZAdd(key, redis.Z{
		Score:  score,
		Member: string(value),
	})

	if result.Err() != nil {
		Log.Error.Print("Redis: ", result.Err())
		prod.Drop(msg)
	}
}

func (prod *Redis) storeString(msg core.Message) {
	value, key := prod.getValueAndKey(msg)

	result := prod.client.Set(key, string(value), 0)
	if result.Err() != nil {
		Log.Error.Print("Redis: ", result.Err())
		prod.Drop(msg)
	}
}

func (prod *Redis) close() {
	defer prod.WorkerDone()
	prod.CloseMessageChannel(prod.store)
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
		Log.Error.Print("Redis: ", err)
	}

	prod.AddMainWorker(workers)
	prod.MessageControlLoop(prod.store)
}
