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

package consumer

import (
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
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
	fileBufferGrowSize = 1024
	fileOffsetStart    = "Start"
	fileOffsetEnd      = "End"
	fileOffsetContinue = "Current"
)

type fileState int32

const (
	fileStateOpen = fileState(iota)
	fileStateRead = fileState(iota)
	fileStateDone = fileState(iota)
)

// File consumer plugin
// This consumer can be paused.
// Configuration example
//
//   - "consumer.File":
//     Enable: true
//     File: "test.txt"
//     Offset: "Current"
//     Delimiter: "\n"
//
// File is a mandatory setting and contains the file to read. The file will be
// read from beginning to end and the reader will stay attached until the
// consumer is stopped. This means appends to the file will be recognized by
// gollum. Symlinks are always resolved, i.e. changing the symlink target will
// be ignored unless gollum is restarted.
//
// Offset defines where to start reading the file. Valid values (case sensitive)
// are "Start", "End", "Current". By default this is set to "End". If "Current"
// is used a filed in /tmp will be created that contains the last position that
// has been read.
//
// Delimiter defines the end of a message inside the file. By default this is
// set to "\n".
type File struct {
	core.ConsumerBase
	file             *os.File
	fileName         string
	continueFileName string
	delimiter        string
	seek             int
	seekOffset       int64
	persistSeek      bool
	state            fileState
}

func init() {
	shared.RuntimeType.Register(File{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *File) Configure(conf core.PluginConfig) error {
	err := cons.ConsumerBase.Configure(conf)
	if err != nil {
		return err
	}

	if !conf.HasValue("File") {
		return core.NewConsumerError("No file configured for consumer.File")
	}

	escapeChars := strings.NewReplacer("\\n", "\n", "\\r", "\r", "\\t", "\t")

	cons.file = nil
	cons.fileName = conf.GetString("File", "")
	cons.delimiter = escapeChars.Replace(conf.GetString("Delimiter", "\n"))
	cons.persistSeek = false

	switch conf.GetString("Offset", fileOffsetEnd) {
	default:
		fallthrough
	case fileOffsetEnd:
		cons.seek = 2
		cons.seekOffset = 0

	case fileOffsetStart:
		cons.seek = 1
		cons.seekOffset = 0

	case fileOffsetContinue:
		cons.seek = 1
		cons.seekOffset = 0
		cons.persistSeek = true
	}

	return nil
}

func (cons *File) postAndPersist(data []byte, sequence uint64) {
	cons.seekOffset, _ = cons.file.Seek(0, 1)
	cons.PostData(data, sequence)
	ioutil.WriteFile(cons.continueFileName, []byte(strconv.FormatInt(cons.seekOffset, 10)), 0644)
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
	}

	if cons.persistSeek {
		baseFileName := cons.realFileName()
		pathDelimiter := strings.NewReplacer("/", "_", ".", "_")
		cons.continueFileName = "/tmp/gollum" + pathDelimiter.Replace(baseFileName) + ".idx"
		cons.seekOffset = 0

		fileContents, err := ioutil.ReadFile(cons.continueFileName)
		if err == nil {
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

	sendFunction := cons.PostData
	if cons.persistSeek {
		sendFunction = cons.postAndPersist
	}

	buffer := shared.NewBufferedReader(fileBufferGrowSize, 0, cons.delimiter, sendFunction)
	printFileOpenError := true

	for cons.state != fileStateDone {
		if cons.IsPaused() {
			runtime.Gosched()
			continue // ### continue, do nothing ###
		}

		// Initialize the seek state if requested
		// Try to read the remains of the file first
		if cons.state == fileStateOpen {
			if cons.file != nil {
				buffer.Read(cons.file)
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
					Log.Error.Print("File open error - ", err)
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
			err := buffer.Read(cons.file)

			switch {
			case err == nil: // ok
			case err == io.EOF:
				runtime.Gosched()
			case cons.state == fileStateRead:
				Log.Error.Print("Error reading file - ", err)
				cons.file.Close()
				cons.file = nil
			}
		}
	}
}

func (cons File) onRoll() {
	cons.setState(fileStateOpen)
}

// Consume listens to stdin.
func (cons *File) Consume(workers *sync.WaitGroup) {
	cons.setState(fileStateOpen)
	defer cons.setState(fileStateDone)

	go func() {
		defer shared.RecoverShutdown()
		cons.AddMainWorker(workers)
		cons.read()
	}()

	cons.DefaultControlLoop(cons.onRoll)
}
