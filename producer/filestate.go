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
	"compress/gzip"
	"fmt"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tio"
	"github.com/trivago/tgo/tlog"
	"github.com/trivago/tgo/tsync"
	"io"
	"os"
	"sync"
	"time"
)

type fileState struct {
	file         *os.File
	bgWriter     *sync.WaitGroup
	batch        core.MessageBatch
	buffer       []byte
	assembly     core.WriterAssembly
	fileCreated  time.Time
	flushTimeout time.Duration
	log          tlog.LogScope
}

type fileRotateConfig struct {
	timeout  time.Duration
	sizeByte int64
	atHour   int
	atMinute int
	zeroPad  int
	enabled  bool
	compress bool
}

func newFileState(maxMessageCount int, modulator core.Modulator, tryFallback func(*core.Message), timeout time.Duration, logScope tlog.LogScope) *fileState {
	return &fileState{
		batch:        core.NewMessageBatch(maxMessageCount),
		bgWriter:     new(sync.WaitGroup),
		flushTimeout: timeout,
		assembly:     core.NewWriterAssembly(nil, tryFallback, modulator),
		log:          logScope,
	}
}

func (state *fileState) flush() {
	if state.file != nil {
		state.assembly.SetWriter(state.file)
		state.batch.Flush(state.assembly.Write)
	} else {
		state.batch.Flush(state.assembly.Flush)
	}
}

func (state *fileState) close() {
	if state.file != nil {
		state.assembly.SetWriter(state.file)
		state.batch.Close(state.assembly.Write, state.flushTimeout)
	} else {
		state.batch.Close(state.assembly.Flush, state.flushTimeout)
	}
	state.bgWriter.Wait()
}

func (state *fileState) compressAndCloseLog(sourceFile *os.File) {
	state.bgWriter.Add(1)
	defer state.bgWriter.Done()

	// Generate file to zip into
	sourceFileName := sourceFile.Name()
	sourceDir, sourceBase, _ := tio.SplitPath(sourceFileName)

	targetFileName := fmt.Sprintf("%s/%s.gz", sourceDir, sourceBase)

	targetFile, err := os.OpenFile(targetFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		state.log.Error.Print("Compress error:", err)
		sourceFile.Close()
		return
	}

	// Create zipfile and compress data
	state.log.Note.Print("Compressing " + sourceFileName)

	sourceFile.Seek(0, 0)
	targetWriter := gzip.NewWriter(targetFile)
	spin := tsync.NewSpinner(tsync.SpinPriorityHigh)

	for err == nil {
		_, err = io.CopyN(targetWriter, sourceFile, 1<<20) // 1 MB chunks
		spin.Yield()                                       // Be async!
	}

	// Cleanup
	sourceFile.Close()
	targetWriter.Close()
	targetFile.Close()

	if err != nil && err != io.EOF {
		state.log.Warning.Print("Compression failed:", err)
		err = os.Remove(targetFileName)
		if err != nil {
			state.log.Error.Print("Compressed file remove failed:", err)
		}
		return
	}

	// Remove original log
	err = os.Remove(sourceFileName)
	if err != nil {
		state.log.Error.Print("Uncompressed file remove failed:", err)
	}
}

func (state *fileState) pruneByHour(baseFilePath string, hours int) {
	state.bgWriter.Wait()
	baseDir, baseName, _ := tio.SplitPath(baseFilePath)

	files, err := tio.ListFilesByDateMatching(baseDir, baseName+".*")
	if err != nil {
		state.log.Error.Print("Error pruning files: ", err)
		return // ### return, error ###
	}

	pruneDate := time.Now().Add(time.Duration(-hours) * time.Hour)

	for i := 0; i < len(files) && files[i].ModTime().Before(pruneDate); i++ {
		filePath := fmt.Sprintf("%s/%s", baseDir, files[i].Name())
		if err := os.Remove(filePath); err != nil {
			state.log.Error.Printf("Failed to prune \"%s\": %s", filePath, err.Error())
		} else {
			state.log.Note.Printf("Pruned \"%s\"", filePath)
		}
	}
}

func (state *fileState) pruneByCount(baseFilePath string, count int) {
	state.bgWriter.Wait()
	baseDir, baseName, _ := tio.SplitPath(baseFilePath)

	files, err := tio.ListFilesByDateMatching(baseDir, baseName+".*")
	if err != nil {
		state.log.Error.Print("Error pruning files: ", err)
		return // ### return, error ###
	}

	numFilesToPrune := len(files) - count
	if numFilesToPrune < 1 {
		return // ## return, nothing to prune ###
	}

	for i := 0; i < numFilesToPrune; i++ {
		filePath := fmt.Sprintf("%s/%s", baseDir, files[i].Name())
		if err := os.Remove(filePath); err != nil {
			state.log.Error.Printf("Failed to prune \"%s\": %s", filePath, err.Error())
		} else {
			state.log.Note.Printf("Pruned \"%s\"", filePath)
		}
	}
}

func (state *fileState) pruneToSize(baseFilePath string, maxSize int64) {
	state.bgWriter.Wait()
	baseDir, baseName, _ := tio.SplitPath(baseFilePath)

	files, err := tio.ListFilesByDateMatching(baseDir, baseName+".*")
	if err != nil {
		state.log.Error.Print("Error pruning files: ", err)
		return // ### return, error ###
	}

	totalSize := int64(0)
	for _, file := range files {
		totalSize += file.Size()
	}

	for _, file := range files {
		if totalSize <= maxSize {
			return // ### return, done ###
		}
		filePath := fmt.Sprintf("%s/%s", baseDir, file.Name())
		if err := os.Remove(filePath); err != nil {
			state.log.Error.Printf("Failed to prune \"%s\": %s", filePath, err.Error())
		} else {
			state.log.Note.Printf("Pruned \"%s\"", filePath)
			totalSize -= file.Size()
		}
	}
}

func (state *fileState) needsRotate(rotate fileRotateConfig, forceRotate bool) (bool, error) {
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
	if !rotate.enabled {
		return false, nil
	}

	if forceRotate {
		return true, nil
	}

	// File is too large?
	if stats.Size() >= rotate.sizeByte {
		return true, nil // ### return, too large ###
	}

	// File is too old?
	if time.Since(state.fileCreated) >= rotate.timeout {
		return true, nil // ### return, too old ###
	}

	// RotateAt crossed?
	if rotate.atHour > -1 && rotate.atMinute > -1 {
		now := time.Now()
		rotateAt := time.Date(now.Year(), now.Month(), now.Day(), rotate.atHour, rotate.atMinute, 0, 0, now.Location())

		if state.fileCreated.Sub(rotateAt).Minutes() < 0 {
			return true, nil // ### return, too old ###
		}
	}

	// nope, everything is ok
	return false, nil
}
