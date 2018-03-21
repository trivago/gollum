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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/components"
	"github.com/trivago/gollum/producer/file"
)

// File producer plugin
//
// The file producer writes messages to a file. This producer also allows log
// rotation and compression of the rotated logs. Folders in the file path will
// be created if necessary.
//
// Each target file will handled with separated batch processing.
//
// Parameters
//
// - File: This value contains the path to the log file to write. The wildcard character "*"
// can be used as a placeholder for the stream name.
// By default this parameter is set to "/var/log/gollum.log".
//
// - FileOverwrite: This value causes the file to be overwritten instead of appending new data
// to it.
// By default this parameter is set to "false".
//
// - Permissions: Defines the UNIX filesystem permissions used when creating
// the named file as an octal number.
// By default this paramater is set to "0664".
//
// - FolderPermissions: Defines the UNIX filesystem permissions used when creating
// the folders as an octal number.
// By default this paramater is set to "0755".
//
// Examples
//
// This example will write the messages from all streams to `/tmp/gollum.log`
// after every 64 message or after 60sec:
//
//  fileOut:
//    Type: producer.File
//    Streams: "*"
//    File: /tmp/gollum.log
//    Batch:
//      MaxCount: 128
//      FlushCount: 64
//      TimeoutSec: 60
//      FlushTimeoutSec: 3
type File struct {
	core.DirectProducer `gollumdoc:"embed_type"`

	// Rotate is public to make Pruner.Configure() callable (bug in treflect package)
	// Prune is public to make FileRotateConfig.Configure() callable (bug in treflect package)
	// BatchConfig is public to make BatchedWriterConfig.Configure() callable (bug in treflect package)
	Rotate      components.RotateConfig        `gollumdoc:"embed_type"`
	Pruner      file.Pruner                    `gollumdoc:"embed_type"`
	BatchConfig components.BatchedWriterConfig `gollumdoc:"embed_type"`

	batchedFileGuard  *sync.RWMutex
	filesByStream     map[core.MessageStreamID]*components.BatchedWriterAssembly // mapped files by stream
	files             map[string]*components.BatchedWriterAssembly               // unique files by target path
	fileDir           string
	fileName          string
	fileExt           string
	filePermissions   os.FileMode `config:"Permissions" default:"0644"`
	folderPermissions os.FileMode `config:"FolderPermissions" default:"0755"`
	overwriteFile     bool        `config:"FileOverwrite"`
	wildcardPath      bool
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

	logFile := conf.GetString("File", "/var/log/gollum.log")
	prod.wildcardPath = strings.IndexByte(logFile, '*') != -1

	prod.fileDir = filepath.Dir(logFile)
	prod.fileExt = filepath.Ext(logFile)
	prod.fileName = filepath.Base(logFile)
	prod.fileName = prod.fileName[:len(prod.fileName)-len(prod.fileExt)]

	prod.batchedFileGuard = new(sync.RWMutex)
}

// Produce writes to a buffer that is dumped to a file.
func (prod *File) Produce(workers *sync.WaitGroup) {
	prod.AddMainWorker(workers)
	prod.TickerMessageControlLoop(prod.writeMessage, prod.BatchConfig.BatchTimeout, prod.writeBatchOnTimeOut)
}

func (prod *File) getBatchedFile(streamID core.MessageStreamID) (*components.BatchedWriterAssembly, error) {
	// get batchedFile from filesByStream[streamID] map
	prod.batchedFileGuard.RLock()
	batchedFile, fileIsLinked := prod.filesByStream[streamID]
	prod.batchedFileGuard.RUnlock()
	if fileIsLinked {
		if rotate, err := batchedFile.NeedsRotate(prod.Rotate, false); !rotate {
			return batchedFile, err // ### return, already open or error ###
		}
	}

	prod.batchedFileGuard.Lock()
	defer prod.batchedFileGuard.Unlock()

	// check again to avoid race conditions
	batchedFile, fileIsLinked = prod.filesByStream[streamID]
	if fileIsLinked {
		if rotate, err := batchedFile.NeedsRotate(prod.Rotate, false); !rotate {
			return batchedFile, err // ### return, already open or error ###
		}
	}

	streamTargetFile := prod.newStreamTargetFile(streamID)

	// get batchedFile from files[path] and assure the file is correctly mapped
	batchedFile, fileExists := prod.files[streamTargetFile.GetOriginalPath()]
	if !fileExists {
		batchedFile = components.NewBatchedWriterAssembly(
			prod.BatchConfig,
			prod,
			prod.TryFallback,
			prod.Logger,
		)

		prod.files[streamTargetFile.GetOriginalPath()] = batchedFile
		prod.filesByStream[streamID] = batchedFile
	} else if !fileIsLinked {
		// in this case two streams target the same file
		// need to link and check rotation again
		prod.filesByStream[streamID] = batchedFile
		if rotate, err := batchedFile.NeedsRotate(prod.Rotate, false); !rotate {
			return batchedFile, err // ### return, already open or error ###
		}
	}

	err := prod.rotateBatchedFile(batchedFile, streamTargetFile)

	return batchedFile, err
}

func (prod *File) rotateBatchedFile(batchedFile *components.BatchedWriterAssembly, streamTargetFile file.TargetFile) error {
	// Assure directory is existing
	if _, err := streamTargetFile.GetDir(); err != nil {
		return err // ### return, missing directory ###
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
		return err // ### return error ###
	}

	batchedFile.SetWriter(fileWriter)

	// Create "current" symlink
	if prod.Rotate.Enabled {
		prod.createCurrentSymlink(finalPath, streamTargetFile.GetSymlinkPath())
	}

	// Prune old logs if requested
	go prod.Pruner.Prune(streamTargetFile.GetOriginalPath())

	return nil
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
	prod.batchedFileGuard.Lock()
	defer prod.batchedFileGuard.Unlock()

	// create unique batchedFile map from prod.filesByStream
	flushMap := map[file.TargetFile]*components.BatchedWriterAssembly{}
	for streamID, batchedFile := range prod.filesByStream {
		streamTargetFile := prod.newStreamTargetFile(streamID)
		flushMap[streamTargetFile] = batchedFile
	}

	// rotate every unique batchedFile
	for streamTargetFile, batchedFile := range flushMap {
		prod.rotateBatchedFile(batchedFile, streamTargetFile)
	}
}

func (prod *File) writeBatchOnTimeOut() {
	for _, batchedFile := range prod.files {
		batchedFile.FlushOnTimeOut()
	}
}

func (prod *File) writeMessage(msg *core.Message) {
	batchedFile, err := prod.getBatchedFile(msg.GetStreamID())
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
