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
	"fmt"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// File producer plugin
// Configuration example
//
//   - "producer.File":
//     Enable: true
//     File: "/var/log/gollum.log"
//     BatchMaxCount: 16384
//     BatchFlushCount: 10000
//     BatchTimeoutSec: 2
//     FlushTimeoutSec: 10
//     Rotate: false
//     RotateTimeoutMin: 1440
//     RotateSizeMB: 1024
//     RotateAt: "00:00"
//     RotateTimestamp: "2006-01-02_15"
//     Compress: true
//
// The file producer writes messages to a file. This producer also allows log
// rotation and compression of the rotated logs. Folders in the file path will
// be created if necessary.
//
// File contains the path to the log file to write. The wildcard character "*"
// can be used as a placeholder for the stream name.
// By default this is set to /var/prod/gollum.log.
//
// BatchMaxCount defines the maximum number of messages that can be buffered
// before a flush is mandatory. If the buffer is full and a flush is still
// underway or cannot be triggered out of other reasons, the producer will
// block.
//
// BatchFlushCount defines the number of messages to be buffered before they are
// written to disk. This setting is clamped to BatchMaxCount.
// By default this is set to BatchMaxCount / 2.
//
// BatchTimeoutSec defines the maximum number of seconds to wait after the last
// message arrived before a batch is flushed automatically. By default this is
// set to 5..
//
// FlushTimeoutSec sets the maximum number of seconds to wait before a flush is
// aborted during shutdown. By default this is set to 0, which does not abort
// the flushing procedure.
//
// Rotate if set to true the logs will rotate after reaching certain thresholds.
//
// RotateTimeoutMin defines a timeout in minutes that will cause the logs to
// rotate. Can be set in parallel with RotateSizeMB. By default this is set to
// 1440 (i.e. 1 Day).
//
// RotateAt defines specific timestamp as in "HH:MM" when the log should be
// rotated. Hours must be given in 24h format. When left empty this setting is
// ignored. By default this setting is disabled.
//
// RotateTimestamp sets the timestamp added to the filename when file rotation
// is enabled. The format is based on Go's time.Format function and set to
// "2006-01-02_15" by default.
//
// RotatePruneCount removes old logfiles upon rotate so that only the given
// number of logfiles remain. Logfiles are located by the name defined by "File"
// and are pruned by date (followed by name).
// By default this is set to 0 which disables pruning.
//
// RotatePruneTotalSizeMB removes old logfiles upon rotate so that only the
// given number of MBs are used by logfiles. Logfiles are located by the name
// defined by "File" and are pruned by date (followed by name).
// By default this is set to 0 which disables pruning.
//
// Compress defines if a rotated logfile is to be gzip compressed or not.
// By default this is set to false.
type File struct {
	core.ProducerBase
	filesByStream   map[core.MessageStreamID]*fileState
	files           map[string]*fileState
	rotate          fileRotateConfig
	timestamp       string
	fileDir         string
	fileName        string
	fileExt         string
	batchTimeout    time.Duration
	flushTimeout    time.Duration
	batchMaxCount   int
	batchFlushCount int
	pruneCount      int
	pruneSize       int64
	wildcardPath    bool
}

func init() {
	shared.RuntimeType.Register(File{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *File) Configure(conf core.PluginConfig) error {
	err := prod.ProducerBase.Configure(conf)
	if err != nil {
		return err
	}

	prod.SetRollCallback(prod.rotateLog)

	prod.filesByStream = make(map[core.MessageStreamID]*fileState)
	prod.files = make(map[string]*fileState)
	prod.batchMaxCount = conf.GetInt("BatchMaxCount", 8192)
	prod.batchFlushCount = conf.GetInt("BatchFlushCount", prod.batchMaxCount/2)
	prod.batchFlushCount = shared.MinI(prod.batchFlushCount, prod.batchMaxCount)
	prod.batchTimeout = time.Duration(conf.GetInt("BatchTimeoutSec", 5)) * time.Second

	logFile := conf.GetString("File", "/var/prod/gollum.log")
	prod.wildcardPath = strings.IndexByte(logFile, '*') != -1

	prod.fileDir = filepath.Dir(logFile)
	prod.fileExt = filepath.Ext(logFile)
	prod.fileName = filepath.Base(logFile)
	prod.fileName = prod.fileName[:len(prod.fileName)-len(prod.fileExt)]
	prod.timestamp = conf.GetString("RotateTimestamp", "2006-01-02_15")
	prod.flushTimeout = time.Duration(conf.GetInt("FlushTimeoutSec", 5)) * time.Second

	prod.rotate.enabled = conf.GetBool("Rotate", false)
	prod.rotate.timeout = time.Duration(conf.GetInt("RotateTimeoutMin", 1440)) * time.Minute
	prod.rotate.sizeByte = int64(conf.GetInt("RotateSizeMB", 1024)) << 20
	prod.rotate.atHour = -1
	prod.rotate.atMinute = -1
	prod.rotate.compress = conf.GetBool("Compress", false)

	prod.pruneCount = conf.GetInt("RotatePruneCount", 0)
	prod.pruneSize = int64(conf.GetInt("RotatePruneTotalSizeMB", 0)) << 20

	if prod.pruneSize > 0 && prod.rotate.sizeByte > 0 {
		prod.pruneSize -= prod.rotate.sizeByte >> 20
		if prod.pruneSize <= 0 {
			prod.pruneCount = 1
			prod.pruneSize = 0
		}
	}

	rotateAt := conf.GetString("RotateAt", "")
	if rotateAt != "" {
		parts := strings.Split(rotateAt, ":")
		rotateAtHour, _ := strconv.ParseInt(parts[0], 10, 8)
		rotateAtMin, _ := strconv.ParseInt(parts[1], 10, 8)

		prod.rotate.atHour = int(rotateAtHour)
		prod.rotate.atMinute = int(rotateAtMin)
	}

	return nil
}

func (prod *File) getFileState(streamID core.MessageStreamID, forceRotate bool) (*fileState, error) {
	if state, stateExists := prod.filesByStream[streamID]; stateExists {
		if rotate, err := state.needsRotate(prod.rotate, forceRotate); !rotate {
			return state, err // ### return, already open or error ###
		}
	}

	var logFileName, fileDir, fileName, fileExt string

	if prod.wildcardPath {
		// Get state from filename (without timestamp, etc.)
		var streamName string
		switch streamID {
		case core.WildcardStreamID:
			streamName = "ALL"
		default:
			streamName = core.StreamTypes.GetStreamName(streamID)
		}

		fileDir = strings.Replace(prod.fileDir, "*", streamName, -1)
		fileName = strings.Replace(prod.fileName, "*", streamName, -1)
		fileExt = strings.Replace(prod.fileExt, "*", streamName, -1)
	} else {
		// Simple case: only one file used
		fileDir = prod.fileDir
		fileName = prod.fileName
		fileExt = prod.fileExt
	}

	logFileBasePath := fmt.Sprintf("%s/%s%s", fileDir, fileName, fileExt)

	// Assure the file is correctly mapped
	state, stateExists := prod.files[logFileBasePath]
	if !stateExists {
		// state does not yet exist: create and map it
		state = newFileState(prod.batchMaxCount, prod.GetFormatter(), prod.Drop, prod.flushTimeout)
		prod.files[logFileBasePath] = state
		prod.filesByStream[streamID] = state
	} else if _, mappingExists := prod.filesByStream[streamID]; !mappingExists {
		// state exists but is not mapped: map it and see if we need to rotate
		prod.filesByStream[streamID] = state
		if rotate, err := state.needsRotate(prod.rotate, forceRotate); !rotate {
			return state, err // ### return, already open or error ###
		}
	}

	// Assure path is existing
	if err := os.MkdirAll(fileDir, 0755); err != nil {
		return nil, fmt.Errorf("Failed to create %s because of %s", fileDir, err.Error()) // ### return, missing directory ###
	}

	// Generate the log filename based on rotation, existing files, etc.
	if !prod.rotate.enabled {
		logFileName = fmt.Sprintf("%s%s", fileName, fileExt)
	} else {
		timestamp := time.Now().Format(prod.timestamp)
		signature := fmt.Sprintf("%s_%s", fileName, timestamp)
		maxSuffix := uint64(0)

		files, _ := ioutil.ReadDir(fileDir)
		for _, file := range files {
			if strings.HasPrefix(file.Name(), signature) {
				counter, _ := shared.Btoi([]byte(file.Name()[len(signature)+1:]))
				if maxSuffix <= counter {
					maxSuffix = counter + 1
				}
			}
		}

		if maxSuffix == 0 {
			logFileName = fmt.Sprintf("%s%s", signature, fileExt)
		} else {
			logFileName = fmt.Sprintf("%s_%d%s", signature, int(maxSuffix), fileExt)
		}
	}

	logFilePath := fmt.Sprintf("%s/%s", fileDir, logFileName)

	// Close existing log
	if state.file != nil {
		currentLog := state.file
		state.file = nil

		if prod.rotate.compress {
			go state.compressAndCloseLog(currentLog)
		} else {
			Log.Note.Print("Rotated ", currentLog.Name(), " -> ", logFilePath)
			currentLog.Close()
		}
	}

	// (Re)open logfile
	var err error
	state.file, err = os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return state, err // ### return error ###
	}

	// Create "current" symlink
	state.fileCreated = time.Now()
	if prod.rotate.enabled {
		symLinkName := fmt.Sprintf("%s/%s_current%s", fileDir, fileName, fileExt)
		os.Remove(symLinkName)
		os.Symlink(logFileName, symLinkName)
	}

	// Prune old logs if requested
	switch {
	case prod.pruneCount > 0 && prod.pruneSize > 0:
		go func() {
			state.pruneByCount(logFileBasePath, prod.pruneCount)
			state.pruneToSize(logFileBasePath, prod.pruneSize)
		}()

	case prod.pruneCount > 0:
		go state.pruneByCount(logFileBasePath, prod.pruneCount)

	case prod.pruneSize > 0:
		go state.pruneToSize(logFileBasePath, prod.pruneSize)
	}

	return state, err
}

func (prod *File) rotateLog() {
	for streamID := range prod.filesByStream {
		if _, err := prod.getFileState(streamID, true); err != nil {
			Log.Error.Print("File rotate error: ", err)
		}
	}
}

func (prod *File) writeBatchOnTimeOut() {
	for _, state := range prod.files {
		if state.batch.ReachedTimeThreshold(prod.batchTimeout) || state.batch.ReachedSizeThreshold(prod.batchFlushCount) {
			state.flush()
		}
	}
}

func (prod *File) writeMessage(msg core.Message) {
	data, streamID := prod.ProducerBase.Format(msg)
	state, err := prod.getFileState(streamID, false)
	if err != nil {
		Log.Error.Print("File log error: ", err)
		prod.Drop(msg)
		return // ### return, dropped ###
	}

	formattedMsg := msg
	formattedMsg.Data = data
	formattedMsg.StreamID = streamID

	if !state.batch.Append(formattedMsg) {
		state.flush()
		if !state.batch.AppendOrBlock(formattedMsg) {
			prod.Drop(msg)
		}
	}
}

// Close gracefully
func (prod *File) Close() {
	defer prod.WorkerDone()

	// Flush buffers
	if prod.CloseGracefully(prod.writeMessage) {
		for _, state := range prod.files {
			state.close()
			state.flush()
			state.waitForFlush()
		}
	}

	// Drop all data that is still in the buffer
	for _, state := range prod.files {
		state.close()
		if !state.batch.IsEmpty() {
			state.flushAndDrop()
			state.waitForFlush()
		}
	}
}

// Produce writes to a buffer that is dumped to a file.
func (prod *File) Produce(workers *sync.WaitGroup) {
	prod.AddMainWorker(workers)
	prod.TickerControlLoop(prod.batchTimeout, prod.writeMessage, prod.writeBatchOnTimeOut)
}
