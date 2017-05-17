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
	"github.com/trivago/tgo/tmath"
	"io"
	"sync"
	"time"
)

// InfluxDB producer plugin
//
// This producer writes data to an influxDB cluster. The data is expected to be
// of a valid influxDB format. As the data format changed between influxDB
// versions it is advisable to use a formatter for the specific influxDB version
// you want to write to. There are collectd to influxDB formatters available
// that can be used (as an example).
//
// Configuration example
//
//  - "producer.InfluxDB":
//    Host: "localhost:8086"
//    User: ""
//    Password: ""
//    Database: "default"
//    TimeBasedName: true
//    UseVersion08: false
//    Version: 100
//    RetentionPolicy: ""
//    BatchMaxCount: 8192
//    BatchFlushCount: 4096
//    BatchTimeoutSec: 5
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
// TimeBasedName enables using time.Format based formatting of databse names.
// I.e. you can use something like "metrics-2006-01-02" to switch databases for
// each day. This setting is enabled by default.
//
// RetentionPolicy correlates to the InfluxDB retention policy setting.
// This is left empty by default (no retention policy used)
//
// UseVersion08 has to be set to true when writing data to InfluxDB 0.8.x.
// By default this is set to false. DEPRECATED. Use Version instead.
//
// Version defines the InfluxDB version to use as in Mmp (Major, minor, patch).
// For version 0.8.x use 80, for version 0.9.0 use 90, for version 1.0.0 use
// use 100 and so on. Defaults to 100.
//
// BatchMaxCount defines the maximum number of messages that can be buffered
// before a flush is mandatory. If the buffer is full and a flush is still
// underway or cannot be triggered out of other reasons, the producer will
// block. By default this is set to 8192.
//
// BatchFlushCount defines the number of messages to be buffered before they are
// written to InfluxDB. This setting is clamped to BatchMaxCount.
// By default this is set to BatchMaxCount / 2.
//
// BatchTimeoutSec defines the maximum number of seconds to wait after the last
// message arrived before a batch is flushed automatically. By default this is
// set to 5.
type InfluxDB struct {
	core.BufferedProducer `gollumdoc:"embed_type"`
	writer          influxDBWriter
	assembly        core.WriterAssembly
	batch           core.MessageBatch
	batchTimeout    time.Duration
	batchMaxCount   int
	batchFlushCount int
}

type influxDBWriter interface {
	io.Writer
	configure(core.PluginConfigReader, *InfluxDB) error
	isConnectionUp() bool
}

func init() {
	core.TypeRegistry.Register(InfluxDB{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *InfluxDB) Configure(conf core.PluginConfigReader) error {
	prod.BufferedProducer.Configure(conf)
	prod.SetStopCallback(prod.close)

	version := conf.GetInt("Version", 100)
	if conf.GetBool("UseVersion08", false) {
		version = 80
	}

	switch {
	case version < 90:
		prod.Log.Debug.Print("Using InfluxDB 0.8.x format")
		prod.writer = new(influxDBWriter08)
	case version == 90:
		prod.Log.Debug.Print("Using InfluxDB 0.9.0 format")
		prod.writer = new(influxDBWriter09)
	default:
		prod.Log.Debug.Print("Using InfluxDB 0.9.1+ format")
		prod.writer = new(influxDBWriter10)
	}

	if err := prod.writer.configure(conf, prod); err != nil {
		return err
	}

	prod.batchMaxCount = conf.GetInt("Batch/MaxCount", 8192)
	prod.batchFlushCount = conf.GetInt("Batch/FlushCount", prod.batchMaxCount/2)
	prod.batchFlushCount = tmath.MinI(prod.batchFlushCount, prod.batchMaxCount)
	prod.batchTimeout = time.Duration(conf.GetInt("Batch/TimeoutSec", 5)) * time.Second

	prod.batch = core.NewMessageBatch(prod.batchMaxCount)
	prod.assembly = core.NewWriterAssembly(prod.writer, prod.TryFallback, prod)
	return conf.Errors.OrNil()
}

// Flush flushes the content of the buffer into the influxdb
func (prod *InfluxDB) sendBatch() {
	if prod.writer.isConnectionUp() {
		prod.batch.Flush(prod.assembly.Write)
	} else if prod.IsStopping() {
		prod.batch.Flush(prod.assembly.Flush)
	}
}

// Threshold based flushing
func (prod *InfluxDB) sendBatchOnTimeOut() {
	if prod.batch.ReachedTimeThreshold(prod.batchTimeout) || prod.batch.ReachedSizeThreshold(prod.batchFlushCount) {
		prod.sendBatch()
	}
}

func (prod *InfluxDB) bufferMessage(msg *core.Message) {
	prod.batch.AppendOrFlush(msg, prod.sendBatch, prod.IsActiveOrStopping, prod.TryFallback)
}

func (prod *InfluxDB) close() {
	defer prod.WorkerDone()

	prod.DefaultClose()
	prod.batch.Close(prod.assembly.Write, prod.GetShutdownTimeout())
}

// Produce starts a bulk producer which will collect datapoints until either the buffer is full or a timeout has been reached.
// The buffer limit does not describe the number of messages received from kafka but the size of the buffer content in KB.
func (prod *InfluxDB) Produce(workers *sync.WaitGroup) {
	prod.AddMainWorker(workers)
	prod.TickerMessageControlLoop(prod.bufferMessage, prod.batchTimeout, prod.sendBatchOnTimeOut)
}
