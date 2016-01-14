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
	"github.com/trivago/tgo"
	"io/ioutil"
	"os"
	"path/filepath"
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
//     BatchTimeoutSec: 5
//     MaxFileSizeMB: 512
//     MaxFileAgeMin: 1
//     MessageSizeByte: 8192
//     RespoolDelaySec: 10
//
// The Spooling producer buffers messages and sends them again to the previous
// stream stored in the message. This means the message must have been routed
// at least once before reaching the spooling producer. If the previous and
// current stream is identical the message is dropped.
// The Formatter configuration value is forced to "format.Serialize" and
// cannot be changed.
// This producer does not implement a fuse breaker.
//
// Path sets the output directory for spooling files. Spooling files will
// Files will be stored as "<path>/<stream>/<number>.spl". By default this is
// set to "/var/run/gollum/spooling".
//
// BatchMaxCount defines the maximum number of messages stored in memory before
// a write to file is triggered. Set to 100 by default.
//
// BatchTimeoutSec defines the maximum number of seconds to wait after the last
// message arrived before a batch is flushed automatically. By default this is
// set to 5.
//
// MaxFileSizeMB sets the size in MB when a spooling file is rotated. Reading
// will start only after a file is rotated. Set to 512 MB by default.
//
// MaxFileAgeMin defines the time in minutes after a spooling file is rotated.
// Reading will start only after a file is rotated. This setting divided by two
// will be used to define the wait time for reading, too.
// Set to 1 minute by default.
//
// BufferSizeByte defines the initial size of the buffer that is used to parse
// messages from a spool file. If a message is larger than this size, the buffer
// will be resized. By default this is set to 8192.
//
// RespoolDelaySec sets the number of seconds to wait before trying to load
// existing spool files after a restart. This is useful for configurations that
// contain dynamic streams. By default this is set to 10.
type Spooling struct {
	core.ProducerBase
	outfile         map[core.MessageStreamID]*spoolFile
	rotation        fileRotateConfig
	path            string
	maxFileSize     int64
	RespoolDuration time.Duration
	maxFileAge      time.Duration
	batchTimeout    time.Duration
	batchMaxCount   int
	bufferSizeByte  int
}

const (
	spoolingMetricWrite    = "Spooling:Write-"
	spoolingMetricRead     = "Spooling:Read-"
	spoolingMetricWriteSec = "Spooling:WriteSec-"
	spoolingMetricReadSec  = "Spooling:ReadSec-"
)

func init() {
	core.TypeRegistry.Register(Spooling{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *Spooling) Configure(conf core.PluginConfig) error {
	conf.Override("Formatter", "format.Serialize")
	err := prod.ProducerBase.Configure(conf)
	if err != nil {
		return err
	}
	prod.SetStopCallback(prod.close)

	prod.path = conf.GetString("Path", "/var/run/gollum/spooling")

	prod.maxFileSize = int64(conf.GetInt("MaxFileSizeMB", 512)) << 20
	prod.maxFileAge = time.Duration(conf.GetInt("MaxFileAgeMin", 1)) * time.Minute
	prod.batchMaxCount = conf.GetInt("BatchMaxCount", 100)
	prod.batchTimeout = time.Duration(conf.GetInt("BatchTimeoutSec", 5)) * time.Second
	prod.outfile = make(map[core.MessageStreamID]*spoolFile)
	prod.RespoolDuration = time.Duration(conf.GetInt("RespoolDelaySec", 10)) * time.Second
	prod.bufferSizeByte = conf.GetInt("BufferSizeByte", 8192)
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

func (prod *Spooling) writeBatchOnTimeOut() {
	for _, spool := range prod.outfile {
		read, write := spool.getAndResetCounts()
		duration := time.Since(spool.lastMetricUpdate)
		spool.lastMetricUpdate = time.Now()

		tgo.Metric.Add(spoolingMetricRead+spool.streamName, read)
		tgo.Metric.Add(spoolingMetricWrite+spool.streamName, write)
		tgo.Metric.SetF(spoolingMetricReadSec+spool.streamName, float64(read)/duration.Seconds())
		tgo.Metric.SetF(spoolingMetricWriteSec+spool.streamName, float64(write)/duration.Seconds())

		if spool.batch.ReachedSizeThreshold(prod.batchMaxCount/2) || spool.batch.ReachedTimeThreshold(prod.batchTimeout) {
			spool.flush()
		}
	}
}

func (prod *Spooling) writeToFile(msg core.Message) {
	// Get the correct file state for this stream
	streamID := msg.PrevStreamID
	spool, exists := prod.outfile[streamID]
	if !exists {
		streamName := core.StreamRegistry.GetStreamName(streamID)
		spool = newSpoolFile(prod, streamName, msg.Source)
		prod.outfile[streamID] = spool

		if err := os.MkdirAll(spool.basePath, 0700); err != nil {
			prod.Log.Error.Printf("Spooling: Failed to create %s because of %s", spool.basePath, err.Error())
			prod.Drop(msg)
			return // ### return, cannot write ###
		}
	}

	// Open/rotate file if nnecessary
	if !spool.openOrRotate() {
		prod.routeToOrigin(msg)
		return // ### return, could not spool to disk ###
	}

	// Append to buffer
	spool.batch.AppendOrFlush(msg, spool.flush, prod.IsActiveOrStopping, prod.Drop)
	spool.countWrite()
}

func (prod *Spooling) routeToOrigin(msg core.Message) {
	msg.Route(msg.PrevStreamID)

	if spool, exists := prod.outfile[msg.PrevStreamID]; exists {
		spool.countRead()
	}
}

func (prod *Spooling) close() {
	defer prod.WorkerDone()

	// Drop as the producer accepting these messages is already offline anyway
	prod.CloseMessageChannel(prod.Drop)
	for _, spool := range prod.outfile {
		spool.close()
	}
}

func (prod *Spooling) openExistingFiles() {
	prod.Log.Debug.Print("Looking for spool files to read")
	files, _ := ioutil.ReadDir(prod.path)
	for _, file := range files {
		if file.IsDir() {
			streamName := filepath.Base(file.Name())
			streamID := core.GetStreamID(streamName)

			// Only create a new spooler if the stream is registered by this instance
			if _, exists := prod.outfile[streamID]; !exists && core.StreamRegistry.IsStreamRegistered(streamID) {
				prod.Log.Note.Printf("Found existing spooling folders for %s", streamName)
				prod.outfile[streamID] = newSpoolFile(prod, streamName, nil)
			}
		}
	}
}

// Produce writes to stdout or stderr.
func (prod *Spooling) Produce(workers *sync.WaitGroup) {
	prod.AddMainWorker(workers)
	time.AfterFunc(prod.RespoolDuration, prod.openExistingFiles)
	prod.TickerMessageControlLoop(prod.writeToFile, prod.batchTimeout, prod.writeBatchOnTimeOut)
}
