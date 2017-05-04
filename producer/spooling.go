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
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo"
)

// Spooling producer plugin
// The Spooling producer buffers messages and sends them again to the previous
// stream stored in the message. This means the message must have been routed
// at least once before reaching the spooling producer. If the previous and
// current stream is identical the message is dropped.
// The Formatter configuration value is forced to "format.Serialize" and
// cannot be changed.
// This producer does not implement a fuse breaker.
// Configuration example
//
//  - "producer.Spooling":
//    Path: "/var/run/gollum/spooling"
//    BatchMaxCount: 100
//    BatchTimeoutSec: 5
//    MaxFileSizeMB: 512
//    MaxFileAgeMin: 1
//    MessageSizeByte: 8192
//    RespoolDelaySec: 10
//    MaxMessagesSec: 100
//    RevertStreamOnDrop: false
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
//
// MaxMessagesSec sets the maximum number of messages that can be respooled per
// second. By default this is set to 100. Setting this value to 0 will cause
// respooling to work as fast as possible.
//
// RevertStreamOnDrop can be used to revert the message stream before dropping
// the message. This can be useful if you e.g. want to write messages that
// could not be spooled to stream separated files on disk. Set to false by
// default.
type Spooling struct {
	core.BufferedProducer
	outfile         map[core.MessageStreamID]*spoolFile
	outfileGuard    *sync.RWMutex
	rotation        fileRotateConfig
	path            string
	maxFileSize     int64
	respoolDuration time.Duration
	maxFileAge      time.Duration
	batchTimeout    time.Duration
	readDelay       time.Duration
	batchMaxCount   int
	bufferSizeByte  int
	revertOnDrop    bool
	spoolCheck      *time.Timer
	serialze        core.Formatter
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
func (prod *Spooling) Configure(conf core.PluginConfigReader) error {
	prod.BufferedProducer.Configure(conf)
	prod.SetPrepareStopCallback(prod.waitForReader)
	prod.SetStopCallback(prod.close)
	prod.SetRollCallback(prod.onRoll)

	serializePlugin, err := core.NewPluginWithConfig(core.NewPluginConfig("", "format.Serialize"))
	conf.Errors.Push(err)
	if serializeFormatter, isFormatter := serializePlugin.(core.Formatter); isFormatter {
		prod.serialze = serializeFormatter
	} else {
		conf.Errors.Pushf("Failed to instantiate format.Serialize")
	}

	prod.path = conf.GetString("Path", "/var/run/gollum/spooling")

	prod.maxFileSize = int64(conf.GetInt("MaxFileSizeMB", 512)) << 20
	prod.maxFileAge = time.Duration(conf.GetInt("MaxFileAgeMin", 1)) * time.Minute
	prod.batchMaxCount = conf.GetInt("Batch/MaxCount", 100)
	prod.batchTimeout = time.Duration(conf.GetInt("Batch/TimeoutSec", 5)) * time.Second
	prod.outfile = make(map[core.MessageStreamID]*spoolFile)
	prod.respoolDuration = time.Duration(conf.GetInt("RespoolDelaySec", 10)) * time.Second
	prod.bufferSizeByte = conf.GetInt("BufferSizeByte", 8192)
	prod.outfileGuard = new(sync.RWMutex)
	prod.revertOnDrop = conf.GetBool("RevertStreamOnDrop", false)

	if maxMsgSec := time.Duration(conf.GetInt("MaxMessagesSec", 100)); maxMsgSec > 0 {
		prod.readDelay = time.Second / maxMsgSec
	} else {
		prod.readDelay = 0
	}

	prod.rotation = fileRotateConfig{
		timeout:  prod.maxFileAge,
		sizeByte: prod.maxFileSize,
		atHour:   -1,
		atMinute: -1,
		enabled:  true,
		compress: false,
	}

	return conf.Errors.OrNil()
}

func (prod *Spooling) Modulate(msg *core.Message) core.ModulateResult {
	result := prod.BufferedProducer.Modulate(msg)
	prod.serialze.ApplyFormatter(msg) // Ignore result
	return result
}

// Drop reverts the message stream before dropping
func (prod *Spooling) Drop(msg *core.Message) {
	if prod.revertOnDrop {
		msg.SetStreamID(msg.PreviousStreamID())
	}
	prod.BufferedProducer.Drop(msg)
}

func (prod *Spooling) writeToFile(msg *core.Message) {
	// Get the correct file state for this stream
	streamID := msg.PreviousStreamID()

	prod.outfileGuard.RLock()
	spool, exists := prod.outfile[streamID]
	prod.outfileGuard.RUnlock()

	if !exists {
		streamName := core.StreamRegistry.GetStreamName(streamID)
		spool = newSpoolFile(prod, streamName, msg.Source())

		prod.outfileGuard.Lock()
		// Recheck to avoid races
		if spool, exists = prod.outfile[streamID]; !exists {
			spool = newSpoolFile(prod, streamName, msg.Source())
			prod.outfile[streamID] = spool
		}
		prod.outfileGuard.Unlock()
	}

	if err := os.MkdirAll(spool.basePath, 0700); err != nil && !os.IsExist(err) {
		prod.Log.Error.Printf("Spooling: Failed to create %s because of %s", spool.basePath, err.Error())
		prod.Drop(msg)
		return // ### return, cannot write ###
	}

	// Open/rotate file if nnecessary
	if !spool.openOrRotate(false) {
		prod.Drop(msg)
		return // ### return, could not spool to disk ###
	}

	// Append to buffer
	spool.batch.AppendOrFlush(msg, spool.flush, prod.IsActiveOrStopping, prod.Drop)
	spool.countWrite()
}

func (prod *Spooling) flush(force bool) {
	prod.outfileGuard.RLock()
	outfiles := prod.outfile
	prod.outfileGuard.RUnlock()

	for _, spool := range outfiles {
		read, write := spool.getAndResetCounts()

		tgo.Metric.Add(spoolingMetricRead+spool.streamName, read)
		tgo.Metric.Add(spoolingMetricWrite+spool.streamName, write)

		if force || spool.batch.ReachedSizeThreshold(prod.batchMaxCount/2) || spool.batch.ReachedTimeThreshold(prod.batchTimeout) {
			spool.flush()
		}
		spool.openOrRotate(force)
	}
}

func (prod *Spooling) writeBatchOnTimeOut() {
	prod.flush(false)
}

func (prod *Spooling) onRoll() {
	prod.flush(true)

	prod.outfileGuard.RLock()
	outfiles := prod.outfile
	prod.outfileGuard.RUnlock()

	for _, file := range outfiles {
		file.triggerRoll()
	}
}

func (prod *Spooling) routeToOrigin(msg *core.Message) {
	routeStart := time.Now()

	msg.SetStreamID(msg.PreviousStreamID()) // Force PrevStreamID to be preserved in case message gets spooled again
	stream := core.StreamRegistry.GetRouter(msg.StreamID())
	core.Route(msg, stream)

	prod.outfileGuard.RLock()
	if spool, exists := prod.outfile[msg.PreviousStreamID()]; exists {
		spool.countRead()
	}
	prod.outfileGuard.RUnlock()

	delay := prod.readDelay - time.Now().Sub(routeStart)
	if delay > 0 {
		time.Sleep(delay)
	}
}

func (prod *Spooling) waitForReader() {
	prod.outfileGuard.Lock()
	defer prod.outfileGuard.Unlock()

	outfiles := prod.outfile
	for _, spool := range outfiles {
		spool.waitForReader()
	}
}

func (prod *Spooling) close() {
	defer prod.WorkerDone()

	// Drop as the producer accepting these messages is already offline anyway
	if prod.spoolCheck != nil {
		prod.spoolCheck.Stop()
	}
	prod.DefaultClose()

	prod.outfileGuard.Lock()
	defer prod.outfileGuard.Unlock()
	for _, spool := range prod.outfile {
		spool.close()
	}
}

// As we might share spooling folders with different instances we only read
// streams that we actually care about.
func (prod *Spooling) openExistingFiles() {
	prod.Log.Debug.Print("Looking for spool files to read")
	files, _ := ioutil.ReadDir(prod.path)
	for _, file := range files {
		if file.IsDir() {
			streamName := filepath.Base(file.Name())
			streamID := core.StreamRegistry.GetStreamID(streamName)

			// Only create a new spooler if the stream is registered by this instance
			prod.outfileGuard.Lock()
			if _, exists := prod.outfile[streamID]; !exists && core.StreamRegistry.IsStreamRegistered(streamID) {
				prod.Log.Note.Printf("Found existing spooling folders for %s", streamName)
				prod.outfile[streamID] = newSpoolFile(prod, streamName, nil)
			}
			prod.outfileGuard.Unlock()
		}
	}

	// Keep looking for new streams
	if prod.IsActive() {
		prod.spoolCheck = time.AfterFunc(prod.respoolDuration, prod.openExistingFiles)
	}
}

// Produce writes to stdout or stderr.
func (prod *Spooling) Produce(workers *sync.WaitGroup) {
	prod.AddMainWorker(workers)
	prod.spoolCheck = time.AfterFunc(prod.respoolDuration, prod.openExistingFiles)
	prod.TickerMessageControlLoop(prod.writeToFile, prod.batchTimeout, prod.writeBatchOnTimeOut)
}
