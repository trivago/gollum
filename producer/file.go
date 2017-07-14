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
	"errors"
	"fmt"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/components"
	"github.com/trivago/gollum/producer/file"
	"github.com/trivago/tgo/tmath"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// File producer plugin
//
// The file producer writes messages to a file. This producer also allows log
// rotation and compression of the rotated logs. Folders in the file path will
// be created if necessary.
//
// Configuration example
//
//  myProducer:
//    Type: producer.File
//    File: "/var/log/gollum.log"
//    FileOverwrite: false
//    Permissions: "0664"
//    FolderPermissions: "0755"
//    Batch:
// 		MaxCount: 8192
//    	FlushCount: 4096
//    	TimeoutSec: 5
//    FlushTimeoutSec: 5
//    Rotation:
//		Enable: false
// 		Timestamp: 2006-01-02_15
//    	TimeoutMin: 1440
//    	SizeMB: 1024
// 		Compress: false
// 		ZeroPadding: 0
// 		At: 13:05
// 	  Prune:
//    	Count: 0
//    	AfterHours: 0
//    	TotalSizeMB: 0
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
// Batch/MaxCount defines the maximum number of messages that can be buffered
// before a flush is mandatory. If the buffer is full and a flush is still
// underway or cannot be triggered out of other reasons, the producer will
// block. By default this is set to 8192.
//
// Batch/FlushCount defines the number of messages to be buffered before they are
// written to disk. This setting is clamped to BatchMaxCount.
// By default this is set to BatchMaxCount / 2.
//
// Batch/TimeoutSec defines the maximum number of seconds to wait after the last
// message arrived before a batch is flushed automatically. By default this is
// set to 5.
//
// FlushTimeoutSec sets the maximum number of seconds to wait before a flush is
// aborted during shutdown. By default this is set to 0, which does not abort
// the flushing procedure.
//
type File struct {
	core.DirectProducer `gollumdoc:"embed_type"`

	// Rotate is public to make Pruner.Configure() callable (bug in treflect package)
	// Prune is public to make FileRotateConfig.Configure() callable (bug in treflect package)
	Rotate components.RotateConfig `gollumdoc:"embed_type"`
	Pruner file.Pruner             `gollumdoc:"embed_type"`

	// configuration
	batchTimeout      time.Duration `config:"Batch/TimeoutSec" default:"5" metric:"sec"`
	batchMaxCount     int           `config:"Batch/MaxCount" default:"8192"`
	batchFlushCount   int           `config:"Batch/FlushCount" default:"4096"`
	flushTimeout      time.Duration `config:"FlushTimeoutSec" default:"0" metric:"sec"`
	overwriteFile     bool          `config:"FileOverwrite"`
	filePermissions   os.FileMode   `config:"Permissions" default:"0644"`
	folderPermissions os.FileMode   `config:"FolderPermissions" default:"0755"`

	// properties
	filesByStream map[core.MessageStreamID]*components.BatchedWriterAssembly
	files         map[string]*components.BatchedWriterAssembly
	fileDir       string
	fileName      string
	fileExt       string
	wildcardPath  bool
}

func init() {
	core.TypeRegistry.Register(File{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *File) Configure(conf core.PluginConfigReader) {
	prod.Pruner.Logger = prod.Logger

	prod.SetRollCallback(prod.rotateLog)
	prod.SetStopCallback(prod.close)

	prod.filesByStream = make(map[core.MessageStreamID]*components.BatchedWriterAssembly)
	prod.files = make(map[string]*components.BatchedWriterAssembly)
	prod.batchFlushCount = tmath.MinI(prod.batchFlushCount, prod.batchMaxCount)

	logFile := conf.GetString("File", "/var/log/gollum.log")
	prod.wildcardPath = strings.IndexByte(logFile, '*') != -1

	prod.fileDir = filepath.Dir(logFile)
	prod.fileExt = filepath.Ext(logFile)
	prod.fileName = filepath.Base(logFile)
	prod.fileName = prod.fileName[:len(prod.fileName)-len(prod.fileExt)]
}

// Produce writes to a buffer that is dumped to a file.
func (prod *File) Produce(workers *sync.WaitGroup) {
	prod.AddMainWorker(workers)
	prod.TickerMessageControlLoop(prod.writeMessage, prod.batchTimeout, prod.writeBatchOnTimeOut)
}

func (prod *File) getBatchedFile(streamID core.MessageStreamID, forceRotate bool) (*components.BatchedWriterAssembly, error) {
	var err error

	// get batchedFile from filesByStream[streamID] map
	if batchedFile, fileExists := prod.filesByStream[streamID]; fileExists {
		if rotate, err := prod.needsRotate(batchedFile, forceRotate); !rotate {
			return batchedFile, err // ### return, already open or error ###
		}
	}

	streamTargetFile := prod.newStreamTargetFile(streamID)

	// get batchedFile from files[path] and assure the file is correctly mapped
	batchedFile, fileExists := prod.files[streamTargetFile.GetOriginalPath()]
	if !fileExists {
		// batchedFile does not yet exist: create and map it
		batchedFile = components.NewBatchedWriterAssembly(
			prod.batchMaxCount,
			prod.batchTimeout,
			prod.batchFlushCount,
			prod,
			prod.TryFallback,
			prod.flushTimeout,
			prod.Logger,
		)

		prod.files[streamTargetFile.GetOriginalPath()] = batchedFile
		prod.filesByStream[streamID] = batchedFile
	} else if _, mappingExists := prod.filesByStream[streamID]; !mappingExists {
		// batchedFile exists but is not mapped: map it and see if we need to Rotate
		prod.filesByStream[streamID] = batchedFile
		if rotate, err := prod.needsRotate(batchedFile, forceRotate); !rotate {
			return batchedFile, err // ### return, already open or error ###
		}
	}

	// Assure directory is existing
	if _, err = streamTargetFile.GetDir(); err != nil {
		return nil, err // ### return, missing directory ###
	}

	finalPath := streamTargetFile.GetFinalPath(prod.Rotate)

	// Close existing batchedFile.writer
	if batchedFile.HasWriter() {
		currentLog := batchedFile.GetWriterAndUnset()

		prod.Logger.Info("Rotated ", currentLog.Name(), " -> ", finalPath)
		go currentLog.Close() // close in subroutine for eventually compression in the background
	}

	// Update BatchedWriterAssembly writer and creation time
	fileWriter, err := prod.newFileStateWriterDisk(finalPath)
	if err != nil {
		return batchedFile, err // ### return error ###
	}

	batchedFile.SetWriter(fileWriter)

	// Create "current" symlink
	if prod.Rotate.Enabled {
		prod.createCurrentSymlink(finalPath, streamTargetFile.GetSymlinkPath())
	}

	// Prune old logs if requested
	go prod.Pruner.Prune(streamTargetFile.GetOriginalPath())

	return batchedFile, err
}

// NeedsRotate evaluate if the BatchedWriterAssembly need to rotate by the FileRotateConfig
func (prod *File) needsRotate(batchFile *components.BatchedWriterAssembly, forceRotate bool) (bool, error) {
	// File does not exist?
	if !batchFile.HasWriter() {
		return true, nil
	}

	// File can be accessed?
	if batchFile.GetWriter().IsAccessible() == false {
		return false, errors.New("Can' access file to rotate")
	}

	// File needs rotation?
	if !prod.Rotate.Enabled {
		return false, nil
	}

	if forceRotate {
		return true, nil
	}

	// File is too large?
	if batchFile.GetWriter().Size() >= prod.Rotate.SizeByte {
		return true, nil // ### return, too large ###
	}

	// File is too old?
	if time.Since(batchFile.Created) >= prod.Rotate.Timeout {
		return true, nil // ### return, too old ###
	}

	// RotateAt crossed?
	if prod.Rotate.AtHour > -1 && prod.Rotate.AtMinute > -1 {
		now := time.Now()
		rotateAt := time.Date(now.Year(), now.Month(), now.Day(), prod.Rotate.AtHour, prod.Rotate.AtMinute, 0, 0, now.Location())

		if batchFile.Created.Sub(rotateAt).Minutes() < 0 {
			return true, nil // ### return, too old ###
		}
	}

	// nope, everything is ok
	return false, nil
}

func (prod *File) createCurrentSymlink(source, target string) {
	symLinkNameTemporary := fmt.Sprintf("%s.tmp", target)

	os.Symlink(source, symLinkNameTemporary)
	os.Rename(symLinkNameTemporary, target)
}

func (prod *File) newFileStateWriterDisk(path string) (*file.BatchedFileWriter, error) {
	openFlags := os.O_RDWR | os.O_CREATE | os.O_APPEND
	if prod.overwriteFile {
		openFlags |= os.O_TRUNC
	} else {
		openFlags |= os.O_APPEND
	}

	fileHandler, err := os.OpenFile(path, openFlags, prod.filePermissions)
	if err != nil {
		return nil, err // ### return error ###
	}

	batchedFileWriter := file.NewBatchedFileWriter(fileHandler, prod.Rotate.Compress, prod.Logger)
	return &batchedFileWriter, nil
}

func (prod *File) newStreamTargetFile(streamID core.MessageStreamID) file.TargetFile {
	var fileDir, fileName, fileExt string

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

	return file.NewTargetFile(fileDir, fileName, fileExt, prod.folderPermissions)
}

func (prod *File) rotateLog() {
	for streamID := range prod.filesByStream {
		if _, err := prod.getBatchedFile(streamID, true); err != nil {
			prod.Logger.Error("Rotate error: ", err)
		}
	}
}

func (prod *File) writeBatchOnTimeOut() {
	for _, batchedFile := range prod.files {
		batchedFile.FlushOnTimeOut()
	}
}

func (prod *File) writeMessage(msg *core.Message) {
	batchedFile, err := prod.getBatchedFile(msg.GetOrigStreamID(), false)
	if err != nil {
		prod.Logger.Error("Write error: ", err)
		prod.TryFallback(msg)
		return // ### return, fallback ###
	}

	batchedFile.Batch.AppendOrFlush(msg, batchedFile.Flush, prod.IsActiveOrStopping, prod.TryFallback)
}

func (prod *File) close() {
	defer prod.WorkerDone()

	for _, batchedFile := range prod.files {
		batchedFile.Close()
	}
}
