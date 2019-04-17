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

	metrics "github.com/rcrowley/go-metrics"

	"github.com/trivago/gollum/core"
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
	metricWrite metrics.Counter
	metricRead  metrics.Counter
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
		metricRead:  metrics.NewCounter(),
		metricWrite: metrics.NewCounter(),
	}

	prod.metricsRegistry.Register("read", spool.metricRead)
	prod.metricsRegistry.Register("write", spool.metricWrite)

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
	spool.metricRead.Inc(1)
}

func (spool *spoolFile) countWrite() {
	spool.metricWrite.Inc(1)
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

	size, err := base64.StdEncoding.Decode(decoded, data)
	if err != nil {
		spool.prod.Logger.WithError(err).Error("failed deserialiying spooled message from base64")
		return
	}

	msg, err := core.DeserializeMessage(decoded[:size])
	if err != nil {
		spool.prod.Logger.WithError(err).Error("failed deserializing spooled message from protobuf")
		return
	}

	spool.prod.routeToOrigin(msg)
}

func (spool *spoolFile) waitForReader() {
	spool.readWorker.Wait()
}

func (spool *spoolFile) read() {
	spool.prod.AddWorker()
	defer spool.prod.WorkerDone()

	spool.readWorker.Add(1)
	defer spool.readWorker.Done()

nextFile:
	for !spool.prod.IsStopping() {
		minSuffix, _ := spool.getFileNumbering()

		// Wait for first spooling file to be rolled.

		spoolFileName := fmt.Sprintf(spoolFileFormatString, spool.basePath, minSuffix)
		if minSuffix == 0 || minSuffix > maxSpoolFileNumber || (spool.file != nil && spool.file.Name() == spoolFileName) {
			if minSuffix > maxSpoolFileNumber {
				spool.prod.Logger.Debug("Read sleeps (no file)")
			} else {
				spool.prod.Logger.Debugf("Read waits for %s", spoolFileName)
			}

			spool.prod.WorkerDone() // to avoid being stuck during shutdown
			spool.readWorker.Done()

			retry := time.After(spool.prod.maxFileAge / 2)
			select {
			case <-retry:
			case <-spool.roll:
			}

			spool.prod.AddWorker() // worker done is always called at exit
			spool.readWorker.Add(1)
			continue // ### continue, try again ###
		}

		// Only spool back if target is not busy
		for !spool.prod.IsStopping() && spool.source != nil && spool.source.IsBlocked() {
			time.Sleep(time.Millisecond * 100)
		}

		if spool.prod.IsStopping() {
			return // ### return, stop requested ###
		}

		file, err := os.OpenFile(spoolFileName, os.O_RDONLY, 0600)
		if err != nil {
			spool.prod.Logger.Error("Read open error ", err)
			// TODO: how to mitigate a possible endless loop?
			continue // ### continue, try again ###
		}

		spool.prod.Logger.Debug("Opened ", spoolFileName, " for reading")
		spool.reader.Reset(0)

		var data []byte
		more := true

		for more {
			// read line by line
			data, more, err = spool.reader.ReadOne(file)

			// Any error marks the file as failed but does not delete it, so messages can eventually be recovered
			if err != io.EOF && err != nil {
				spool.prod.Logger.WithError(err).Error("failed to read spooling file ", spoolFileName)
				if err := file.Close(); err != nil {
					spool.prod.Logger.WithError(err).Error("failed to close spooling file ", spoolFileName)
				}

				spool.prod.Logger.Debug("renaming ", spoolFileName)
				os.Rename(spoolFileName, spoolFileName+".failed")
				continue nextFile // ### continue, read error or EOF ###
			}

			// len(data) == 0 may be an incomplete message, i.e. we need to read once more to get the rest
			if len(data) > 0 {
				spool.decode(data)
			}

			// Check if we're forced to stop reading
			if spool.prod.IsStopping() {
				spool.prod.Logger.Warning("file ", spoolFileName, " will be read again after restart")
				if err := file.Close(); err != nil {
					spool.prod.Logger.WithError(err).Error("failed to close spooling file ", spoolFileName)
				}
				return // ### return, stop requested ###
			}
		}

		// Close and remove file as it has been completely read
		if err := file.Close(); err != nil {
			spool.prod.Logger.WithError(err).Error("failed to close spooling file ", spoolFileName)
		}

		spool.prod.Logger.Debug("removing ", spoolFileName)
		os.Remove(spoolFileName)
	}
}
