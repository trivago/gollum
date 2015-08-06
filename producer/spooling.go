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
	"os"
	"sync"
	"time"
)

// Spooling producer plugin
// Configuration example
//
//   - "producer.Spooling":
//     Enable: true
//     Path: "/var/run/gollum/spooling"
//     BatchMaxCount: 100
//     MaxFileSizeMB: 512
//     MaxFileAgeMin: 1
//
// The Spooling producer buffers messages and sends them again to the previous
// stream stored in the message. This means the message must have been routed
// at least once before reaching the spooling producer. If the previous and
// current stream is identical the message is dropped.
// The Formatter configuration value is forced to "format.Serialize" and
// cannot be changed.
//
// Path sets the output directory for spooling files. Spooling files will
// Files will be stored as "<path>/<stream>/<number>.spl". By default this is
// set to "/var/run/gollum/spooling".
//
// BatchMaxCount defines the maximum number of messages stored in memory before
// a write to file is triggered. Set to 100 by default.
//
// MaxFileSizeMB sets the size in MB when a spooling file is rotated. Reading
// will start only after a file is rotated. Set to 512 MB by default.
//
// MaxFileAgeMin defines the time in minutes after a spooling file is rotated.
// Reading will start only after a file is rotated. This setting divided by two
// will be used to define the wait time for reading, too.
// Set to 1 minute by default.
type Spooling struct {
	core.ProducerBase
	outfile       map[core.MessageStreamID]*spoolFile
	rotation      fileRotateConfig
	path          string
	maxFileSize   int64
	maxFileAge    time.Duration
	batchMaxCount int
}

const spoolingMetricName = "Spooling:Write-"
const spooledMetricName = "Spooling:Read-"

func init() {
	shared.TypeRegistry.Register(Spooling{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *Spooling) Configure(conf core.PluginConfig) error {
	conf.Override("Formatter", "format.Serialize")
	err := prod.ProducerBase.Configure(conf)
	if err != nil {
		return err
	}
	prod.SetStopCallback(prod.close)

	prod.path = conf.GetString("Path", "/var/run/gollum/spooling/")

	prod.maxFileSize = int64(conf.GetInt("MaxFileSizeMB", 512)) << 20
	prod.maxFileAge = time.Duration(conf.GetInt("MaxFileAgeMin", 1)) * time.Minute
	prod.batchMaxCount = conf.GetInt("BatchMaxCount", 100)
	prod.outfile = make(map[core.MessageStreamID]*spoolFile)
	prod.rotation = fileRotateConfig{
		timeout:  prod.maxFileAge,
		sizeByte: prod.maxFileSize,
		atHour:   -1,
		atMinute: -1,
		enabled:  true,
		compress: false,
	}

	return nil
}

func (prod *Spooling) writeToFile(msg core.Message) {
	// Get the correct file state for this stream
	spool, exists := prod.outfile[msg.PrevStreamID]
	if !exists {
		streamName := core.StreamRegistry.GetStreamName(msg.PrevStreamID)
		if err := os.MkdirAll(prod.path+"/"+streamName, 0700); err != nil {
			Log.Error.Printf("Failed to create %s because of %s", prod.path, err.Error())
			prod.Drop(msg)
			return // ### return, cannot write ###
		}

		spool = newSpoolFile(prod, streamName)
		prod.outfile[msg.PrevStreamID] = spool
	}

	// Open/rotate file if nnecessary
	if !spool.openOrRotate() {
		prod.routeToOrigin(msg)
		return // ### return, could not spool to disk ###
	}

	// Flush if limits are reached
	if spool.batch.ReachedSizeThreshold(prod.batchMaxCount/2) || spool.batch.ReachedTimeThreshold(prod.maxFileAge) {
		spool.flush()
	}

	// Append to buffer
	spool.batch.AppendRetry(msg, spool.flush, prod.IsActive, prod.Drop)
	shared.Metric.Inc(spoolingMetricName + spool.streamName)
}

func (prod *Spooling) routeToOrigin(msg core.Message) {
	if prod.IsActive() {
		msg.Route(msg.PrevStreamID)
	} else {
		prod.Drop(msg)
	}

	if spool, exists := prod.outfile[msg.PrevStreamID]; exists {
		shared.Metric.Inc(spooledMetricName + spool.streamName)
	}
}

func (prod *Spooling) close() {
	defer prod.WorkerDone()

	// Drop as the producer accepting these messages is already offline anyway
	prod.CloseGracefully(prod.Drop)
	for _, spool := range prod.outfile {
		spool.close()
	}
}

// Produce writes to stdout or stderr.
func (prod *Spooling) Produce(workers *sync.WaitGroup) {
	prod.AddMainWorker(workers)
	prod.MessageControlLoop(prod.writeToFile)
}
