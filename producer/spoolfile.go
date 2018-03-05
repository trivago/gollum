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
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo"
	"github.com/trivago/tgo/tio"
	"github.com/trivago/tgo/tmath"
	"github.com/trivago/tgo/tstrings"
)

type spoolFile struct {
	file        *os.File
	batch       core.MessageBatch
	assembly    core.WriterAssembly
	fileCreated time.Time
	streamName  string
	basePath    string
	prod        *Spooling
	source      core.MessageSource
	readCount   int64
	writeCount  int64
	readWorker  *sync.WaitGroup
	roll        chan struct{}
	reader      *tio.BufferedReader
}

const maxSpoolFileNumber = 99999999 // maximum file number defined by %08d -> 8 digits
const spoolFileFormatString = "%s/%08d.spl"

func newSpoolFile(prod *Spooling, streamName string, source core.MessageSource) *spoolFile {
	spool := &spoolFile{
		file:        nil,
		batch:       core.NewMessageBatch(prod.batchMaxCount),
		assembly:    core.NewWriterAssembly(nil, prod.TryFallback, prod),
		fileCreated: time.Now(),
		streamName:  streamName,
		basePath:    prod.path + "/" + streamName,
		prod:        prod,
		source:      source,
		readWorker:  &sync.WaitGroup{},
		reader:      tio.NewBufferedReader(prod.bufferSizeByte, tio.BufferedReaderFlagDelimiter, 0, "\n"),
		roll:        make(chan struct{}, 1),
	}

	writeMetric := spoolingMetricWrite + streamName
	tgo.Metric.New(writeMetric)
	tgo.Metric.NewRate(writeMetric, spoolingMetricWriteSec+streamName, time.Second, 10, 3, true)

	readMetric := spoolingMetricRead + streamName
	tgo.Metric.New(readMetric)
	tgo.Metric.NewRate(readMetric, spoolingMetricReadSec+streamName, time.Second, 10, 3, true)
	go spool.read()
	return spool
}

func (spool *spoolFile) triggerRoll() {
	spool.roll <- struct{}{}
}

func (spool *spoolFile) flush() {
	spool.batch.Flush(spool.assembly.Write)
}

func (spool *spoolFile) close() {
	for !spool.batch.IsEmpty() {
		spool.batch.Flush(spool.assembly.Write)
		spool.batch.WaitForFlush(spool.prod.GetShutdownTimeout())
	}
	spool.file.Close()
}

func (spool *spoolFile) getAndResetCounts() (read int64, write int64) {
	return atomic.SwapInt64(&spool.readCount, 0), atomic.SwapInt64(&spool.writeCount, 0)
}

func (spool *spoolFile) countRead() {
	atomic.AddInt64(&spool.readCount, 1)
}

func (spool *spoolFile) countWrite() {
	atomic.AddInt64(&spool.writeCount, 1)
}

func (spool *spoolFile) getFileNumbering() (min int, max int) {
	min, max = maxSpoolFileNumber+1, 0
	files, _ := ioutil.ReadDir(spool.basePath)
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".spl" {
			base := filepath.Base(file.Name())
			number, _ := tstrings.Btoi([]byte(base)) // Because we need leading zero support
			min = tmath.MinI(min, int(number))
			max = tmath.MaxI(max, int(number))
		}
	}
	return min, max
}

func (spool *spoolFile) openOrRotate(force bool) bool {
	err := spool.batch.AfterFlushDo(func() error {
		fileSize := int64(0)

		if spool.file != nil {
			fileInfo, err := spool.file.Stat()
			if err != nil {
				return err // ### return, filestat error ###
			}
			fileSize = fileInfo.Size()
		}

		if force || spool.file == nil ||
			fileSize >= spool.prod.maxFileSize ||
			(fileSize > 0 && time.Since(spool.fileCreated) > spool.prod.maxFileAge) {

			_, maxSuffix := spool.getFileNumbering()
			spoolFileName := fmt.Sprintf(spoolFileFormatString, spool.basePath, maxSuffix+1)
			newFile, err := os.OpenFile(spoolFileName, os.O_WRONLY|os.O_CREATE, 0600)
			if err != nil {
				return err // ### return, could not open file ###
			}

			// Set writer and update internal state
			spool.assembly.SetWriter(newFile)

			if spool.file != nil {
				spool.file.Close()
			}

			spool.file = newFile
			spool.fileCreated = time.Now()
			spool.prod.Logger.Debug("Opened ", spoolFileName, " for writing")
		}

		return nil
	})

	if err != nil {
		spool.prod.Logger.Error(err)
		return false // ### return, could not open file ###
	}

	return true
}

func (spool *spoolFile) decode(data []byte) {
	// Base64 decode, than deserialize
	decoded := make([]byte, base64.StdEncoding.DecodedLen(len(data)))

	if size, err := base64.StdEncoding.Decode(decoded, data); err != nil {
		spool.prod.Logger.Error("File read: ", err)
	} else if msg, err := core.DeserializeMessage(decoded[:size]); err != nil {
		spool.prod.Logger.Error("File read: ", err)
	} else {
		spool.prod.routeToOrigin(msg)
	}
}

func (spool *spoolFile) waitForReader() {
	spool.readWorker.Wait()
}

func (spool *spoolFile) read() {
	spool.prod.AddWorker()
	spool.readWorker.Add(1)
	defer spool.prod.WorkerDone()
	defer spool.readWorker.Done()

	for !spool.prod.IsStopping() {
		minSuffix, _ := spool.getFileNumbering()

		spoolFileName := fmt.Sprintf(spoolFileFormatString, spool.basePath, minSuffix)
		if minSuffix == 0 || minSuffix > maxSpoolFileNumber || (spool.file != nil && spool.file.Name() == spoolFileName) {
			if minSuffix > maxSpoolFileNumber {
				spool.prod.Logger.Debug("Read sleeps (no file)")
			} else {
				spool.prod.Logger.Debugf("Read waits for %s", spoolFileName)
			}

			spool.prod.WorkerDone() // to avoid being stuck during shutdown

			retry := time.After(spool.prod.maxFileAge / 2)
			select {
			case <-retry:
			case <-spool.roll:
			}

			spool.prod.AddWorker() // worker done is always called at exit
			continue               // ### continue, try again ###
		}

		file, err := os.OpenFile(spoolFileName, os.O_RDONLY, 0600)
		if err != nil {
			spool.prod.Logger.Error("Read open error ", err)
			continue // ### continue, try again ###
		}

		spool.prod.Logger.Debug("Opened ", spoolFileName, " for reading")
		spool.reader.Reset(0)
		readFailed := false

		for spool.prod.IsStopping() {
			// Only spool back if target is not busy
			if spool.source != nil && spool.source.IsBlocked() {
				time.Sleep(time.Millisecond * 100)
				continue // ### continue, busy source ###
			}

			// Any error cancels the loop
			if err := spool.reader.ReadAll(file, spool.decode); err != nil {
				if err != io.EOF {
					readFailed = true
					spool.prod.Logger.Error("Read error: ", err)
				}
				break // ### break, read error or EOF ###
			}
		}

		// Close and remove file
		spool.prod.Logger.Debug("Removing ", spoolFileName)
		file.Close()
		if readFailed {
			// Rename file for future processing
			spool.prod.Logger.Debug("Renaming ", spoolFileName)
			os.Rename(spoolFileName, spoolFileName+".failed")
		} else {
			// Delete file
			spool.prod.Logger.Debug("Removing ", spoolFileName)
			os.Remove(spoolFileName)
		}
	}
}
