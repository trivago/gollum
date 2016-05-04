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

package consumer

import (
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo"
	"github.com/trivago/tgo/tio"
	"github.com/trivago/tgo/tstrings"
	"github.com/trivago/tgo/tsync"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	fileBufferGrowSize = 1024
	fileOffsetStart    = "oldest"
	fileOffsetEnd      = "newest"
)

type fileState int32

const (
	fileStateOpen = fileState(iota)
	fileStateRead = fileState(iota)
	fileStateDone = fileState(iota)
)

// File consumer plugin
// The file consumer allows to read from files while looking for a delimiter
// that marks the end of a message. If the file is part of e.g. a log rotation
// the file consumer can be set to a symbolic link of the latest file and
// (optionally) be told to reopen the file by sending a SIGHUP. A symlink to
// a file will automatically be reopened if the underlying file is changed.
// When attached to a fuse, this consumer will stop accepting messages in case
// that fuse is burned.
// Configuration example
//
//  - "consumer.File":
//    File: "/var/run/system.log"
//    DefaultOffset: "Newest"
//    OffsetFile: ""
//    Delimiter: "\n"
//
// File is a mandatory setting and contains the file to read. The file will be
// read from beginning to end and the reader will stay attached until the
// consumer is stopped. I.e. appends to the attached file will be recognized
// automatically.
//
// DefaultOffset defines where to start reading the file. Valid values are
// "oldest" and "newest". If OffsetFile is defined the DefaultOffset setting
// will be ignored unless the file does not exist.
// By default this is set to "newest".
//
// OffsetFile defines the path to a file that stores the current offset inside
// the given file. If the consumer is restarted that offset is used to continue
// reading. By default this is set to "" which disables the offset file.
//
// Delimiter defines the end of a message inside the file. By default this is
// set to "\n".
type File struct {
	core.SimpleConsumer
	file           *os.File
	fileName       string
	offsetFileName string
	delimiter      string
	seek           int
	seekOnRotate   int
	seekOffset     int64
	state          fileState
}

func init() {
	core.TypeRegistry.Register(File{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *File) Configure(conf core.PluginConfigReader) error {
	cons.SimpleConsumer.Configure(conf)
	cons.SetRollCallback(cons.onRoll)

	cons.file = nil
	cons.fileName = conf.GetString("File", "/var/run/system.log")
	cons.offsetFileName = conf.GetString("OffsetFile", "")
	cons.delimiter = tstrings.Unescape(conf.GetString("Delimiter", "\n"))

	switch strings.ToLower(conf.GetString("DefaultOffset", fileOffsetEnd)) {
	default:
		fallthrough
	case fileOffsetEnd:
		cons.seek = 2
		cons.seekOnRotate = 2
		cons.seekOffset = 0

	case fileOffsetStart:
		cons.seek = 1
		cons.seekOnRotate = 1
		cons.seekOffset = 0
	}

	return conf.Errors.OrNil()
}

func (cons *File) storeOffset() {
	ioutil.WriteFile(cons.offsetFileName, []byte(strconv.FormatInt(cons.seekOffset, 10)), 0644)
}

func (cons *File) enqueueAndPersist(data []byte) {
	cons.seekOffset, _ = cons.file.Seek(0, 1)
	cons.Enqueue(data)
	cons.storeOffset()
}

func (cons *File) realFileName() string {
	baseFileName, err := filepath.EvalSymlinks(cons.fileName)
	if err != nil {
		baseFileName = cons.fileName
	}

	baseFileName, err = filepath.Abs(baseFileName)
	if err != nil {
		baseFileName = cons.fileName
	}

	return baseFileName
}

func (cons *File) setState(state fileState) {
	cons.state = state
}

func (cons *File) initFile() {
	defer cons.setState(fileStateRead)

	if cons.file != nil {
		cons.file.Close()
		cons.file = nil
		cons.seek = cons.seekOnRotate
		cons.seekOffset = 0
		if cons.offsetFileName != "" {
			cons.storeOffset()
		}
	}

	if cons.offsetFileName != "" {
		fileContents, err := ioutil.ReadFile(cons.offsetFileName)
		if err == nil {
			cons.seek = 1
			cons.seekOffset, err = strconv.ParseInt(string(fileContents), 10, 64)
		}
	}
}

func (cons *File) close() {
	if cons.file != nil {
		cons.file.Close()
	}
	cons.setState(fileStateDone)
	cons.WorkerDone()
}

func (cons *File) read() {
	defer cons.close()

	sendFunction := cons.Enqueue
	if cons.offsetFileName != "" {
		sendFunction = cons.enqueueAndPersist
	}

	spin := tsync.NewSpinner(tsync.SpinPriorityLow)
	buffer := tio.NewBufferedReader(fileBufferGrowSize, 0, 0, cons.delimiter)
	printFileOpenError := true

	for cons.state != fileStateDone {

		// Initialize the seek state if requested
		// Try to read the remains of the file first
		if cons.state == fileStateOpen {
			if cons.file != nil {
				buffer.ReadAll(cons.file, sendFunction)
			}
			cons.initFile()
			buffer.Reset(uint64(cons.seekOffset))
		}

		// Try to open the file to read from
		if cons.state == fileStateRead && cons.file == nil {
			file, err := os.OpenFile(cons.realFileName(), os.O_RDONLY, 0666)

			switch {
			case err != nil:
				if printFileOpenError {
					cons.Log.Warning.Print("Open failed: ", err)
					printFileOpenError = false
				}
				time.Sleep(3 * time.Second)
				continue // ### continue, retry ###

			default:
				cons.file = file
				cons.seekOffset, _ = cons.file.Seek(cons.seekOffset, cons.seek)
				printFileOpenError = true
			}
		}

		// Try to read from the file
		if cons.state == fileStateRead && cons.file != nil {
			err := buffer.ReadAll(cons.file, sendFunction)
			cons.WaitOnFuse()

			switch {
			case err == nil: // ok
				spin.Reset()

			case err == io.EOF:
				if cons.file.Name() != cons.realFileName() {
					cons.Log.Note.Print("Rotation detected")
					cons.onRoll()
				} else {
					newStat, newStatErr := os.Stat(cons.realFileName())
					oldStat, oldStatErr := cons.file.Stat()

					if newStatErr == nil && oldStatErr == nil && !os.SameFile(newStat, oldStat) {
						cons.Log.Note.Print("Rotation detected")
						cons.onRoll()
					}
				}
				spin.Yield()

			case cons.state == fileStateRead:
				cons.Log.Error.Print("Reading failed: ", err)
				cons.file.Close()
				cons.file = nil
			}
		}
	}
}

func (cons *File) onRoll() {
	cons.setState(fileStateOpen)
}

// Consume listens to stdin.
func (cons *File) Consume(workers *sync.WaitGroup) {
	cons.setState(fileStateOpen)
	defer cons.setState(fileStateDone)

	go tgo.WithRecoverShutdown(func() {
		cons.AddMainWorker(workers)
		cons.read()
	})

	cons.ControlLoop()
}
