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
	"fmt"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tmath"
	"github.com/trivago/tgo/tstrings"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// File producer plugin
// The file producer writes messages to a file. This producer also allows log
// rotation and compression of the rotated logs. Folders in the file path will
// be created if necessary.
// This producer does not implement a fuse breaker.
// Configuration example
//
//  - "producer.File":
//    File: "/var/log/gollum.log"
//    FileOverwrite: false
//    Permissions: "0664"
//    FolderPermissions: "0755"
//    BatchMaxCount: 8192
//    BatchFlushCount: 4096
//    BatchTimeoutSec: 5
//    FlushTimeoutSec: 0
//    Rotate: false
//    RotateTimeoutMin: 1440
//    RotateSizeMB: 1024
//    RotateAt: ""
//    RotateTimestamp: "2006-01-02_15"
//    RotatePruneCount: 0
//    RotatePruneAfterHours: 0
//    RotatePruneTotalSizeMB: 0
//    Compress: false
//
// File contains the path to the log file to write. The wildcard character "*"
// can be used as a placeholder for the stream name.
// By default this is set to /var/log/gollum.log.
//
// FileOverwrite enables files to be overwritten instead of appending new data
// to it. This is set to false by default.
//
// Permissions accepts an octal number string that contains the unix file
// permissions used when creating a file. By default this is set to "0664".
//
// FolderPermissions accepts an octal number string that contains the unix file
// permissions used when creating a folder. By default this is set to "0755".
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
//
// FlushTimeoutSec sets the maximum number of seconds to wait before a flush is
// aborted during shutdown. By default this is set to 0, which does not abort
// the flushing procedure.
//
// Rotate if set to true the logs will rotate after reaching certain thresholds.
// By default this is set to false.
//
// RotateTimeoutMin defines a timeout in minutes that will cause the logs to
// rotate. Can be set in parallel with RotateSizeMB. By default this is set to
// 1440 (i.e. 1 Day).
//
// RotateAt defines specific timestamp as in "HH:MM" when the log should be
// rotated. Hours must be given in 24h format. When left empty this setting is
// ignored. By default this setting is disabled.
//
// RotateSizeMB defines the maximum file size in MB that triggers a file rotate.
// Files can get bigger than this size. By default this is set to 1024.
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
// RotatePruneAfterHours removes old logfiles that are older than a given number
// of hours. By default this is set to 0 which disables pruning.
//
// RotatePruneTotalSizeMB removes old logfiles upon rotate so that only the
// given number of MBs are used by logfiles. Logfiles are located by the name
// defined by "File" and are pruned by date (followed by name).
// By default this is set to 0 which disables pruning.
//
// RotateZeroPadding sets the number of leading zeros when rotating files with
// an existing name. Setting this setting to 0 won't add zeros, every other
// number defines the number of leading zeros to be used. By default this is
// set to 0.
//
// Compress defines if a rotated logfile is to be gzip compressed or not.
// By default this is set to false.
type File struct {
	core.BufferedProducer
	filesByStream     map[core.MessageStreamID]*fileState
	files             map[string]*fileState
	rotate            fileRotateConfig
	timestamp         string
	fileDir           string
	fileName          string
	fileExt           string
	batchTimeout      time.Duration
	flushTimeout      time.Duration
	batchMaxCount     int
	batchFlushCount   int
	pruneCount        int
	pruneHours        int
	pruneSize         int64
	wildcardPath      bool
	overwriteFile     bool
	filePermissions   os.FileMode
	folderPermissions os.FileMode
}

func init() {
	core.TypeRegistry.Register(File{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *File) Configure(conf core.PluginConfigReader) error {
	prod.BufferedProducer.Configure(conf)
	prod.SetRollCallback(prod.rotateLog)
	prod.SetStopCallback(prod.close)

	prod.filesByStream = make(map[core.MessageStreamID]*fileState)
	prod.files = make(map[string]*fileState)
	prod.batchMaxCount = conf.GetInt("Batch/MaxCount", 8192)
	prod.batchFlushCount = conf.GetInt("Batch/FlushCount", prod.batchMaxCount/2)
	prod.batchFlushCount = tmath.MinI(prod.batchFlushCount, prod.batchMaxCount)
	prod.batchTimeout = time.Duration(conf.GetInt("Batch/TimeoutSec", 5)) * time.Second
	prod.overwriteFile = conf.GetBool("FileOverwrite", false)

	fileFlags, err := strconv.ParseInt(conf.GetString("Permissions", "0664"), 8, 32)
	conf.Errors.Push(err)
	prod.filePermissions = os.FileMode(fileFlags)

	folderFlags, err := strconv.ParseInt(conf.GetString("FolderPermissions", "0755"), 8, 32)
	conf.Errors.Push(err)
	prod.folderPermissions = os.FileMode(folderFlags)

	logFile := conf.GetString("File", "/var/log/gollum.log")
	prod.wildcardPath = strings.IndexByte(logFile, '*') != -1

	prod.fileDir = filepath.Dir(logFile)
	prod.fileExt = filepath.Ext(logFile)
	prod.fileName = filepath.Base(logFile)
	prod.fileName = prod.fileName[:len(prod.fileName)-len(prod.fileExt)]
	prod.flushTimeout = time.Duration(conf.GetInt("FlushTimeoutSec", 5)) * time.Second

	prod.timestamp = conf.GetString("Rotation/Timestamp", "2006-01-02_15")
	prod.rotate.enabled = conf.GetBool("Rotation/Enable", false)
	prod.rotate.timeout = time.Duration(conf.GetInt("Rotation/TimeoutMin", 1440)) * time.Minute
	prod.rotate.sizeByte = int64(conf.GetInt("Rotation/SizeMB", 1024)) << 20
	prod.rotate.atHour = -1
	prod.rotate.atMinute = -1
	prod.rotate.compress = conf.GetBool("Rotation/Compress", false)
	prod.rotate.zeroPad = conf.GetInt("Rotation/ZeroPadding", 0)

	prod.pruneCount = conf.GetInt("Prune/Count", 0)
	prod.pruneHours = conf.GetInt("Prune/AfterHours", 0)
	prod.pruneSize = int64(conf.GetInt("Prune/TotalSizeMB", 0)) << 20

	if prod.pruneSize > 0 && prod.rotate.sizeByte > 0 {
		prod.pruneSize -= prod.rotate.sizeByte >> 20
		if prod.pruneSize <= 0 {
			prod.pruneCount = 1
			prod.pruneSize = 0
		}
	}

	rotateAt := conf.GetString("Rotation/At", "")
	if rotateAt != "" {
		parts := strings.Split(rotateAt, ":")
		rotateAtHour, _ := strconv.ParseInt(parts[0], 10, 8)
		rotateAtMin, _ := strconv.ParseInt(parts[1], 10, 8)

		prod.rotate.atHour = int(rotateAtHour)
		prod.rotate.atMinute = int(rotateAtMin)
	}

	return conf.Errors.OrNil()
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
			streamName = core.StreamRegistry.GetStreamName(streamID)
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
		state = newFileState(prod.batchMaxCount, prod.Format, prod.Drop, prod.flushTimeout, prod.Log)
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
	if err := os.MkdirAll(fileDir, prod.folderPermissions); err != nil {
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
				// Special case.
				// If there is no extension, counter stays at 0
				// If there is an extension (and no count), parsing the "." will yield a counter of 0
				// If there is a count, parsing it will work as intended
				counter := uint64(0)
				if len(file.Name()) > len(signature) {
					counter, _ = tstrings.Btoi([]byte(file.Name()[len(signature)+1:]))
				}

				if maxSuffix <= counter {
					maxSuffix = counter + 1
				}
			}
		}

		if maxSuffix == 0 {
			logFileName = fmt.Sprintf("%s%s", signature, fileExt)
		} else {
			formatString := "%s_%d%s"
			if prod.rotate.zeroPad > 0 {
				formatString = fmt.Sprintf("%%s_%%0%dd%%s", prod.rotate.zeroPad)
			}
			logFileName = fmt.Sprintf(formatString, signature, int(maxSuffix), fileExt)
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
			prod.Log.Note.Print("Rotated ", currentLog.Name(), " -> ", logFilePath)
			currentLog.Close()
		}
	}

	// (Re)open logfile
	var err error
	openFlags := os.O_RDWR | os.O_CREATE | os.O_APPEND
	if prod.overwriteFile {
		openFlags |= os.O_TRUNC
	} else {
		openFlags |= os.O_APPEND
	}

	state.file, err = os.OpenFile(logFilePath, openFlags, prod.filePermissions)
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
	go func() {
		if prod.pruneHours > 0 {
			state.pruneByHour(logFileBasePath, prod.pruneHours)
		}
		if prod.pruneCount > 0 {
			state.pruneByCount(logFileBasePath, prod.pruneCount)
		}
		if prod.pruneSize > 0 {
			state.pruneToSize(logFileBasePath, prod.pruneSize)
		}
	}()

	return state, err
}

func (prod *File) rotateLog() {
	for streamID := range prod.filesByStream {
		if _, err := prod.getFileState(streamID, true); err != nil {
			prod.Log.Error.Print("Rotate error: ", err)
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

func (prod *File) writeMessage(msg *core.Message) {
	streamMsg := *msg
	prod.BufferedProducer.Format(&streamMsg)
	state, err := prod.getFileState(streamMsg.StreamID(), false)
	if err != nil {
		prod.Log.Error.Print("Write error: ", err)
		prod.Drop(msg)
		return // ### return, dropped ###
	}

	state.batch.AppendOrFlush(msg, state.flush, prod.IsActiveOrStopping, prod.Drop)
}

func (prod *File) close() {
	defer prod.WorkerDone()

	prod.DefaultClose()
	for _, state := range prod.files {
		state.close()
	}
}

// Produce writes to a buffer that is dumped to a file.
func (prod *File) Produce(workers *sync.WaitGroup) {
	prod.AddMainWorker(workers)
	prod.TickerMessageControlLoop(prod.writeMessage, prod.batchTimeout, prod.writeBatchOnTimeOut)
}
