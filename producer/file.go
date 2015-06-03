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
	"compress/gzip"
	"fmt"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	"hash/fnv"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	fileProducerTimestamp = "2006-01-02_15"
)

// File producer plugin
// Configuration example
//
//   - "producer.File":
//     Enable: true
//     File: "/var/log/gollum.log"
//     BatchSizeMaxKB: 16384
//     BatchSizeByte: 4096
//     BatchTimeoutSec: 2
//     Rotate: false
//     RotateTimeoutMin: 1440
//     RotateSizeMB: 1024
//     RotateAt: "00:00"
//     Compress: true
//
// The file producer writes messages to a file. This producer also allows log
// rotation and compression of the rotated logs.
//
// File contains the path to the log file to write. The wildcard "*" will be
// replaced by the stream name.
// By default this is set to /var/prod/gollum.log.
//
// BatchSizeMaxKB defines the internal file buffer size in KB.
// This producers allocates a front- and a backbuffer of this size. If the
// frontbuffer is filled up completely a flush is triggered and the frontbuffer
// becomes available for writing again. Messages larger than BatchSizeMaxKB are
// rejected.
//
// BatchSizeByte defines the number of bytes to be buffered before they are written
// to disk. By default this is set to 8KB.
//
// BatchTimeoutSec defines the maximum number of seconds to wait after the last
// message arrived before a batch is flushed automatically. By default this is
// set to 5..
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
// Compress defines if a rotated logfile is to be gzip compressed or not.
// By default this is set to false.
type File struct {
	core.ProducerBase
	filesByStream    map[core.MessageStreamID]*fileLogState
	files            map[uint32]*fileLogState
	fileDir          string
	fileName         string
	fileExt          string
	wildcardPath     bool
	rotateSizeByte   int64
	bufferSizeMax    int
	batchSize        int
	batchTimeout     time.Duration
	rotateTimeoutMin int
	rotateAtHour     int
	rotateAtMin      int
	rotate           bool
	compress         bool
}

type fileLogState struct {
	file        *os.File
	batch       *core.MessageBatch
	bgWriter    *sync.WaitGroup
	fileCreated time.Time
}

func init() {
	shared.RuntimeType.Register(File{})
}

func newFileLogState(bufferSizeMax int, format core.Formatter) *fileLogState {
	return &fileLogState{
		batch:    core.NewMessageBatch(bufferSizeMax, format),
		bgWriter: new(sync.WaitGroup),
	}
}

// Configure initializes this producer with values from a plugin config.
func (prod *File) Configure(conf core.PluginConfig) error {
	err := prod.ProducerBase.Configure(conf)
	if err != nil {
		return err
	}

	prod.filesByStream = make(map[core.MessageStreamID]*fileLogState)
	prod.files = make(map[uint32]*fileLogState)
	prod.bufferSizeMax = conf.GetInt("BatchSizeMaxKB", 8<<10) << 10 // 8 MB

	prod.batchSize = conf.GetInt("BatchSizeByte", 8192)
	prod.batchTimeout = time.Duration(conf.GetInt("BatchTimeoutSec", 5)) * time.Second

	prod.rotate = conf.GetBool("Rotate", false)
	prod.rotateTimeoutMin = conf.GetInt("RotateTimeoutMin", 1440)
	prod.rotateSizeByte = int64(conf.GetInt("RotateSizeMB", 1024)) << 20
	prod.rotateAtHour = -1
	prod.rotateAtMin = -1
	prod.compress = conf.GetBool("Compress", false)

	logFile := conf.GetString("File", "/var/prod/gollum.log")
	prod.wildcardPath = strings.IndexByte(logFile, '*') != -1

	prod.fileDir = filepath.Dir(logFile)
	prod.fileExt = filepath.Ext(logFile)
	prod.fileName = filepath.Base(logFile)
	prod.fileName = prod.fileName[:len(prod.fileName)-len(prod.fileExt)]

	rotateAt := conf.GetString("RotateAt", "")
	if rotateAt != "" {
		parts := strings.Split(rotateAt, ":")
		rotateAtHour, _ := strconv.ParseInt(parts[0], 10, 8)
		rotateAtMin, _ := strconv.ParseInt(parts[1], 10, 8)

		prod.rotateAtHour = int(rotateAtHour)
		prod.rotateAtMin = int(rotateAtMin)
	}

	return nil
}

func (state *fileLogState) compressAndCloseLog(sourceFile *os.File) {
	state.bgWriter.Add(1)
	defer state.bgWriter.Done()

	// Generate file to zip into
	sourceFileName := sourceFile.Name()
	sourceDir := filepath.Dir(sourceFileName)
	sourceExt := filepath.Ext(sourceFileName)
	sourceBase := filepath.Base(sourceFileName)
	sourceBase = sourceBase[:len(sourceBase)-len(sourceExt)]

	targetFileName := fmt.Sprintf("%s/%s.gz", sourceDir, sourceBase)

	targetFile, err := os.OpenFile(targetFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		Log.Error.Print("File compress error:", err)
		sourceFile.Close()
		return
	}

	// Create zipfile and compress data
	Log.Note.Print("Compressing " + sourceFileName)

	sourceFile.Seek(0, 0)
	targetWriter := gzip.NewWriter(targetFile)

	for err == nil {
		_, err = io.CopyN(targetWriter, sourceFile, 1<<20) // 1 MB chunks
		runtime.Gosched()                                  // Be async!
	}

	// Cleanup
	sourceFile.Close()
	targetWriter.Close()
	targetFile.Close()

	if err != nil && err != io.EOF {
		Log.Warning.Print("Compression failed:", err)
		err = os.Remove(targetFileName)
		if err != nil {
			Log.Error.Print("Compressed file remove failed:", err)
		}
		return
	}

	// Remove original log
	err = os.Remove(sourceFileName)
	if err != nil {
		Log.Error.Print("Uncompressed file remove failed:", err)
	}
}

func (state *fileLogState) onWriterError(err error) bool {
	Log.Error.Print("File write error:", err)
	return false
}

func (state *fileLogState) writeBatch() {
	state.batch.Flush(state.file, nil, state.onWriterError)
}

func (state *fileLogState) needsRotate(prod *File, forceRotate bool) (bool, error) {
	// File does not exist?
	if state.file == nil {
		return true, nil
	}

	// File can be accessed?
	stats, err := state.file.Stat()
	if err != nil {
		return false, err
	}

	// File needs rotation?
	if !prod.rotate {
		return false, nil
	}

	if forceRotate {
		return true, nil
	}

	// File is too large?
	if stats.Size() >= prod.rotateSizeByte {
		return true, nil // ### return, too large ###
	}

	// File is too old?
	if time.Since(state.fileCreated).Minutes() >= float64(prod.rotateTimeoutMin) {
		return true, nil // ### return, too old ###
	}

	// RotateAt crossed?
	if prod.rotateAtHour > -1 && prod.rotateAtMin > -1 {
		now := time.Now()
		rotateAt := time.Date(now.Year(), now.Month(), now.Day(), prod.rotateAtHour, prod.rotateAtMin, 0, 0, now.Location())

		if state.fileCreated.Sub(rotateAt).Minutes() < 0 {
			return true, nil // ### return, too old ###
		}
	}

	// nope, everything is ok
	return false, nil
}

func (prod *File) getFileLogState(streamID core.MessageStreamID, forceRotate bool) (*fileLogState, error) {
	if state, stateExists := prod.filesByStream[streamID]; stateExists {
		if rotate, err := state.needsRotate(prod, forceRotate); !rotate {
			return state, err // ### return, already open or error ###
		}
	}

	var logFileName, fileDir, fileName, fileExt string
	var fileID uint32

	if prod.wildcardPath {
		// Get state from filename (without timestamp, etc.)
		var streamName string
		switch streamID {
		case core.WildcardStreamID:
			streamName = "all"
		case core.LogInternalStreamID:
			streamName = "gollum"
		case core.DroppedStreamID:
			streamName = "dropped"
		default:
			streamName = core.StreamTypes.GetStreamName(streamID)
		}

		fileDir = strings.Replace(prod.fileDir, "*", streamName, -1)
		fileName = strings.Replace(prod.fileName, "*", streamName, -1)
		fileExt = strings.Replace(prod.fileExt, "*", streamName, -1)

		// Hash the base name
		hash := fnv.New32a()
		hash.Write([]byte(fmt.Sprintf("%s/%s%s", fileDir, fileName, fileExt)))
		fileID = hash.Sum32()
	} else {
		// Simple case: only one file used
		fileDir = prod.fileDir
		fileName = prod.fileName
		fileExt = prod.fileExt
		fileID = 0
	}

	// Assure the file is correctly mapped
	state, stateExists := prod.files[fileID]
	if !stateExists {
		// state does not yet exist: create and map it
		state = newFileLogState(prod.bufferSizeMax, prod.ProducerBase.GetFormatter())
		prod.files[fileID] = state
		prod.filesByStream[streamID] = state
	} else if _, mappingExists := prod.filesByStream[streamID]; !mappingExists {
		// state exists but is not mapped: map it and see if we need to rotate
		prod.filesByStream[streamID] = state
		if rotate, err := state.needsRotate(prod, forceRotate); !rotate {
			return state, err // ### return, already open or error ###
		}
	}

	// Generate the log filename based on rotation, existing files, etc.
	if !prod.rotate {
		logFileName = fmt.Sprintf("%s%s", fileName, fileExt)
	} else {
		timestamp := time.Now().Format(fileProducerTimestamp)
		signature := fmt.Sprintf("%s_%s", fileName, timestamp)
		counter := 0

		if err := os.MkdirAll(fileDir, 0755); err != nil {
			Log.Error.Print("Error creating directory " + fileDir)
		}

		files, _ := ioutil.ReadDir(fileDir)
		for _, file := range files {
			if strings.Contains(file.Name(), signature) {
				counter++
			}
		}

		if counter == 0 {
			logFileName = fmt.Sprintf("%s%s", signature, fileExt)
		} else {
			logFileName = fmt.Sprintf("%s_%d%s", signature, counter, fileExt)
		}
	}

	logFile := fmt.Sprintf("%s/%s", fileDir, logFileName)

	// Close existing log
	if state.file != nil {
		currentLog := state.file
		state.file = nil

		if prod.compress {
			go state.compressAndCloseLog(currentLog)
		} else {
			Log.Note.Print("Rotated " + currentLog.Name())
			currentLog.Close()
		}
	}

	// (Re)open logfile
	var err error
	state.file, err = os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return state, err // ### return error ###
	}

	// Create "current" symlink
	state.fileCreated = time.Now()
	if prod.rotate {
		symLinkName := fmt.Sprintf("%s/%s_current", fileDir, fileName)
		os.Remove(symLinkName)
		os.Symlink(logFileName, symLinkName)
	}

	return state, err
}

func (prod *File) writeBatchOnTimeOut() {
	for _, state := range prod.files {
		if state.batch.ReachedTimeThreshold(prod.batchTimeout) || state.batch.ReachedSizeThreshold(prod.batchSize) {
			state.writeBatch()
		}
	}
}

func (prod *File) writeMessage(msg core.Message) {
	state, err := prod.getFileLogState(msg.StreamID, false)
	if err != nil {
		Log.Error.Print("File log error:", err)
		msg.Drop(time.Duration(0))
		return // ### return, dropped ###
	}

	if !state.batch.Append(msg) {
		state.writeBatch()
		state.batch.Append(msg)
	}
}

func (prod *File) rotateLog() {
	for streamID := range prod.filesByStream {
		if _, err := prod.getFileLogState(streamID, true); err != nil {
			Log.Error.Print("File rotate error:", err)
		}
	}
}

func (prod *File) flush() {
	for _, state := range prod.files {
		state.writeBatch()
		state.batch.WaitForFlush(5 * time.Second)

		state.bgWriter.Wait()
		state.file.Close()
	}
	prod.WorkerDone()
}

// Produce writes to a buffer that is dumped to a file.
func (prod *File) Produce(workers *sync.WaitGroup) {
	defer prod.flush()

	prod.AddMainWorker(workers)
	prod.TickerControlLoop(prod.batchTimeout, prod.writeMessage, prod.rotateLog, prod.writeBatchOnTimeOut)
}
