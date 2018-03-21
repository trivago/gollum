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
	"github.com/trivago/gollum/core"
	"io"
	"sync"
)

// InfluxDB producer
//
// This producer writes data to an influxDB endpoint. Data is not converted to
// the correct influxDB format automatically. Proper formatting might be
// required.
//
// Parameters
//
// - Version: Defines the InfluxDB protocol version to use. This can either be
// 80-89 for 0.8.x, 90 for 0.9.0 or 91-100 for 0.9.1 or later.
// Be default this parameter is set to 100.
//
// - Host: Defines the host (and port) of the InfluxDB master.
// Be default this parameter is set to "localhost:8086".
//
// - User: Defines the InfluxDB username to use. If this is empty,
// credentials are not used.
// Be default this parameter is set to "".
//
// - Password: Defines the InfluxDB password to use.
// Be default this parameter is set to "".
//
// - Database: Sets the InfluxDB database to write to.
// Be default this parameter is set to "default".
//
// - TimeBasedName: When set to true, the Database parameter is treated as a
// template for time.Format and the resulting string is used as the database
// name. You can e.g. use "default-2006-01-02" to switch databases each day.
// By default this parameter is set to "true".
//
// - RetentionPolicy: Only available for Version 90. This setting defines the
// InfluxDB retention policy allowed with this protocol version.
// By default this parameter is set to "".
//
// Examples
//
//  metricsToInflux:
//    Type: producer.InfluxDB
//    Streams: metrics
//    Host: "influx01:8086"
//    Database: "metrics"
//    TimeBasedName: false
//    Batch:
//      MaxCount: 2000
//      FlushCount: 100
//      TimeoutSec: 5
type InfluxDB struct {
	core.BatchedProducer `gollumdoc:"embed_type"`
	writer               influxDBWriter
	assembly             core.WriterAssembly
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
func (prod *InfluxDB) Configure(conf core.PluginConfigReader) {
	version := conf.GetInt("Version", 100)

	switch {
	case version < 90:
		prod.Logger.Debug("Using InfluxDB 0.8.x protocol")
		prod.writer = new(influxDBWriter08)
	case version == 90:
		prod.Logger.Debug("Using InfluxDB 0.9.0 protocol")
		prod.writer = new(influxDBWriter09)
	default:
		prod.Logger.Debug("Using InfluxDB 1.0.0 protocol")
		prod.writer = new(influxDBWriter10)
	}

	if err := prod.writer.configure(conf, prod); conf.Errors.Push(err) {
		return
	}

	prod.assembly = core.NewWriterAssembly(prod.writer, prod.TryFallback, prod)
}

// sendBatch returns core.AssemblyFunc to flush batch
func (prod *InfluxDB) sendBatch() core.AssemblyFunc {
	if prod.writer.isConnectionUp() {
		return prod.assembly.Write
	} else if prod.IsStopping() {
		return prod.assembly.Flush
	}

	return nil
}

// Produce starts a bulk producer which will collect datapoints until either the buffer is full or a timeout has been reached.
// The buffer limit does not describe the number of messages received from kafka but the size of the buffer content in KB.
func (prod *InfluxDB) Produce(workers *sync.WaitGroup) {
	prod.BatchMessageLoop(workers, prod.sendBatch)
}
