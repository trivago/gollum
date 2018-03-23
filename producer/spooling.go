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
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/components"
	"github.com/trivago/tgo"
)

// Spooling producer
//
// This producer is meant to be used as a fallback if another producer fails to
// send messages, e.g. because a service is down. It does not really produce
// messages to some other service, it buffers them on disk for a certain time
// and inserts them back to the system after this period.
//
// Parameters
//
// - Path: Sets the output directory for spooling files. Spooling files will
// be stored as "<path>/<stream name>/<number>.spl".
// By default this parameter is set to "/var/run/gollum/spooling".
//
// - MaxFileSizeMB: Sets the size limit in MB that causes a spool file rotation.
// Reading messages back into the system will start only after a file is
// rotated.
// By default this parameter is set to 512.
//
// - MaxFileAgeMin: Defines the duration in minutes after which a spool file
// rotation is triggered (regardless of MaxFileSizeMB). Reading messages back
// into the system will start only after a file is rotated.
// By default this parameter is set to 1.
//
// - MaxMessagesSec: Sets the maximum number of messages that will be respooled
// per second. Setting this value to 0 will cause respooling to send as fast as
// possible.
// By default this parameter is set to 100.
//
// - RespoolDelaySec: Defines the number of seconds to wait before trying to
// load existing spool files from disk after a restart. This setting can be used
// to define a safe timeframe for gollum to set up all required connections and
// resources before putting additionl load on it.
// By default this parameter is set to 10.
//
// - RevertStreamOnFallback: This allows the spooling fallback to handle the
// messages that would have been sent back by the spooler if it would have
// handled the message. When set to true it will revert the stream of the
// message to the previous stream ID before sending it to the Fallback stream.
// By default this parameter is set to false.
//
// - BufferSizeByte: Defines the initial size of the buffer that is used to read
// messages from a spool file. If a message is larger than this size, the buffer
// will be resized.
// By default this parameter is set to 8192.
//
// - Batch/MaxCount: defines the maximum number of messages stored in memory before
// a write to file is triggered.
// By default this parameter is set to 100.
//
// - Batch/TimeoutSec: defines the maximum number of seconds to wait after the last
// message arrived before a batch is flushed automatically.
// By default this parameter is set to 5.
//
// Examples
//
// This example will collect messages from the fallback stream and buffer them
// for 10 minutes. After 10 minutes the first messages will be written back to
// the system as fast as possible.
//
//  spooling:
//    Type: producer.Spooling
//    Stream: fallback
//    MaxMessagesSec: 0
//    MaxFileAgeMin: 10
//
type Spooling struct {
	core.BufferedProducer `gollumdoc:"embed_type"`
	outfile               map[core.MessageStreamID]*spoolFile
	outfileGuard          *sync.RWMutex
	rotation              components.RotateConfig `gollumdoc:"embed_type"`
	path                  string                  `config:"Path" default:"/var/run/gollum/spooling"`
	maxFileSize           int64                   `config:"MaxFileSizeMB" default:"512" metric:"mb"`
	maxFileAge            time.Duration           `config:"MaxFileAgeMin" default:"1" metric:"min"`
	respoolDuration       time.Duration           `config:"RespoolDelaySec" default:"10" metric:"sec"`
	revertOnDrop          bool                    `config:"RevertStreamOnFallback" default:"false"`
	bufferSizeByte        int                     `config:"BufferSizeByte" default:"8192"`
	batchMaxCount         int                     `config:"Batch/MaxCount" default:"100"`
	batchTimeout          time.Duration           `config:"Batch/TimeoutSec" default:"5" metric:"sec"`
	readDelay             time.Duration
	spoolCheck            *time.Timer
	serialze              core.Formatter
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
func (prod *Spooling) Configure(conf core.PluginConfigReader) {
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

	prod.outfileGuard = new(sync.RWMutex)
	prod.outfile = make(map[core.MessageStreamID]*spoolFile)

	if maxMsgSec := time.Duration(conf.GetInt("MaxMessagesSec", 100)); maxMsgSec > 0 {
		prod.readDelay = time.Second / maxMsgSec
	} else {
		prod.readDelay = 0
	}

	//TODO: check if rotation is still in use
	prod.rotation = components.NewRotateConfig()
	prod.rotation.Enabled = true
	prod.rotation.Timeout = prod.maxFileAge
	prod.rotation.SizeByte = prod.maxFileSize
}

// Modulate enforces the serialize formatter at the end of the modulation chain
func (prod *Spooling) Modulate(msg *core.Message) core.ModulateResult {
	result := prod.BufferedProducer.Modulate(msg)
	prod.serialze.ApplyFormatter(msg) // Ignore result
	return result
}

// TryFallback reverts the message stream before dropping
func (prod *Spooling) TryFallback(msg *core.Message) {
	if prod.revertOnDrop {
		msg.SetStreamID(msg.GetPrevStreamID())
	}
	prod.BufferedProducer.TryFallback(msg)
}

func (prod *Spooling) writeToFile(msg *core.Message) {
	// Get the correct file state for this stream
	streamID := msg.GetPrevStreamID()

	prod.outfileGuard.RLock()
	spool, exists := prod.outfile[streamID]
	prod.outfileGuard.RUnlock()

	if !exists {
		streamName := core.StreamRegistry.GetStreamName(streamID)

		prod.outfileGuard.Lock()
		// Recheck to avoid races
		if spool, exists = prod.outfile[streamID]; !exists {
			spool = newSpoolFile(prod, streamName, msg.GetSource())
			prod.outfile[streamID] = spool
		}
		prod.outfileGuard.Unlock()
	}

	if err := os.MkdirAll(spool.basePath, 0700); err != nil && !os.IsExist(err) {
		prod.Logger.Errorf("Spooling: Failed to create %s because of %s", spool.basePath, err.Error())
		prod.TryFallback(msg)
		return // ### return, cannot write ###
	}

	// Open/rotate file if nnecessary
	if !spool.openOrRotate(false) {
		prod.TryFallback(msg)
		return // ### return, could not spool to disk ###
	}

	// Append to buffer
	spool.batch.AppendOrFlush(msg, spool.flush, prod.IsActiveOrStopping, prod.TryFallback)
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

	msg.SetStreamID(msg.GetPrevStreamID()) // Force PrevStreamID to be preserved in case message gets spooled again
	stream := core.StreamRegistry.GetRouter(msg.GetStreamID())
	core.Route(msg, stream)

	prod.outfileGuard.RLock()
	if spool, exists := prod.outfile[msg.GetPrevStreamID()]; exists {
		spool.countRead()
	}
	prod.outfileGuard.RUnlock()

	delay := prod.readDelay - time.Since(routeStart)
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
	prod.Logger.Debug("Looking for spool files to read")
	files, _ := ioutil.ReadDir(prod.path)
	for _, file := range files {
		if file.IsDir() {
			streamName := filepath.Base(file.Name())
			streamID := core.StreamRegistry.GetStreamID(streamName)

			// Only create a new spooler if the stream is registered by this instance
			prod.outfileGuard.Lock()
			if _, exists := prod.outfile[streamID]; !exists && core.StreamRegistry.IsStreamRegistered(streamID) {
				prod.Logger.Infof("Found existing spooling folders for %s", streamName)
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
