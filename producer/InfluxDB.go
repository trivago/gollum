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
	"github.com/trivago/gollum/shared"
	"io"
	"sync"
	"time"
)

// InfluxDB producer plugin
// Configuration example
//
//   - "producer.InfluxDB":
//     Enable: true
//     Host: "localhost:8086"
//     User: ""
//     Password: ""
//     Database: "default"
//     RetentionPolicy: ""
//     BatchMaxCount: 8192
//     BatchFlushCount: 4096
//     BatchTimeoutSec: 5
//
// Host defines the host (and port) of the InfluxDB server.
// Defaults to "localhost:8086".
//
// User defines the InfluxDB username to use to login. If this name is
// left empty credentials are assumed to be disabled. Defaults to empty.
//
// Password defines the user's password. Defaults to empty.
//
// Database sets the InfluxDB database to write to. By default this is
// is set to "default".
//
// RetentionPolicy correlates to the InfluxDB retention policy setting.
// This is left empty by default (no retention policy used)
//
// WriteInflux08 has to be set to true when writing data to InfluxDB 0.8.x.
// By default this is set to false.
//
// BatchMaxCount defines the maximum number of messages that can be buffered
// before a flush is mandatory. If the buffer is full and a flush is still
// underway or cannot be triggered out of other reasons, the producer will
// block. By default this is set to 8192.
//
// BatchFlushCount defines the number of messages to be buffered before they are
// written to disk. This setting is clamped to BatchMaxCount.
// By default this is set to BatchMaxCount / 2.
//
// BatchTimeoutSec defines the maximum number of seconds to wait after the last
// message arrived before a batch is flushed automatically. By default this is
// set to 5.
type InfluxDB struct {
	core.ProducerBase
	writer          influxDBWriter
	assembly        core.WriterAssembly
	batch           core.MessageBatch
	batchTimeout    time.Duration
	flushTimeout    time.Duration
	batchMaxCount   int
	batchFlushCount int
}

type influxDBWriter interface {
	io.Writer
	configure(core.PluginConfig) error
	isConnectionUp() bool
}

func init() {
	shared.TypeRegistry.Register(InfluxDB{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *InfluxDB) Configure(conf core.PluginConfig) error {
	if err := prod.ProducerBase.Configure(conf); err != nil {
		return err
	}

	if conf.GetBool("WriteInflux08", false) {
		prod.writer = new(influxDBWriter08)
	} else {
		prod.writer = new(influxDBWriter09)
	}

	if err := prod.writer.configure(conf); err != nil {
		return err
	}

	prod.flushTimeout = time.Duration(conf.GetInt("BatchTimeoutSeconds", 30)) * time.Second
	prod.batchMaxCount = conf.GetInt("BatchMaxCount", 8192)
	prod.batchFlushCount = conf.GetInt("BatchFlushCount", prod.batchMaxCount/2)
	prod.batchFlushCount = shared.MinI(prod.batchFlushCount, prod.batchMaxCount)
	prod.batchTimeout = time.Duration(conf.GetInt("BatchTimeoutSec", 5)) * time.Second

	prod.batch = core.NewMessageBatch(prod.batchMaxCount)
	prod.assembly = core.NewWriterAssembly(prod.writer, prod.Drop, prod.GetFormatter())
	return nil
}

// Flush flushes the content of the buffer into the influxdb
func (prod *InfluxDB) sendBatch() {
	if prod.writer.isConnectionUp() {
		prod.batch.Flush(prod.assembly.Write)
	}
}

// Threshold based flushing
func (prod *InfluxDB) sendBatchOnTimeOut() {
	if prod.batch.ReachedTimeThreshold(prod.batchTimeout) || prod.batch.ReachedSizeThreshold(prod.batchFlushCount) {
		prod.sendBatch()
	}
}

func (prod *InfluxDB) sendMessage(msg core.Message) {
	if !prod.batch.Append(msg) {
		prod.sendBatch()
		if !prod.batch.AppendOrBlock(msg) {
			prod.Drop(msg)
		}
	}
}

// Close gracefully
func (prod *InfluxDB) Close() {
	defer prod.WorkerDone()

	// Flush buffer to regular socket
	if prod.CloseGracefully(prod.sendMessage) {
		prod.batch.Close()
		prod.sendBatch()
		prod.batch.WaitForFlush(prod.GetShutdownTimeout())
	}

	// Drop all data that is still in the buffer
	if !prod.batch.IsEmpty() {
		prod.batch.Close()
		prod.batch.Flush(prod.assembly.Flush)
		prod.batch.WaitForFlush(prod.GetShutdownTimeout())
	}

}

// Produce starts a bulk producer which will collect datapoints until either the buffer is full or a timeout has been reached.
// The buffer limit does not describe the number of messages received from kafka but the size of the buffer content in KB.
func (prod *InfluxDB) Produce(workers *sync.WaitGroup) {
	prod.AddMainWorker(workers)
	prod.TickerControlLoop(prod.flushTimeout, prod.sendMessage, prod.sendBatchOnTimeOut)
}
