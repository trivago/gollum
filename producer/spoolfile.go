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
	"bufio"
	"encoding/base64"
	"fmt"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo"
	"github.com/trivago/tgo/tlog"
	"github.com/trivago/tgo/tmath"
	"github.com/trivago/tgo/tstrings"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
)

type spoolFile struct {
	file             *os.File
	batch            core.MessageBatch
	assembly         core.WriterAssembly
	fileCreated      time.Time
	streamName       string
	basePath         string
	prod             *Spooling
	source           core.MessageSource
	readCount        int64
	writeCount       int64
	lastMetricUpdate time.Time
	log              tlog.LogScope
}

const maxSpoolFileNumber = 99999999 // maximum file number defined by %08d -> 8 digits
const spoolFileFormatString = "%s/%08d.spl"

func newSpoolFile(prod *Spooling, streamName string, source core.MessageSource) *spoolFile {
	spool := &spoolFile{
		file:             nil,
		batch:            core.NewMessageBatch(prod.batchMaxCount),
		assembly:         core.NewWriterAssembly(nil, prod.Drop, prod.GetFormatter()),
		fileCreated:      time.Now(),
		streamName:       streamName,
		basePath:         prod.path + "/" + streamName,
		prod:             prod,
		source:           source,
		lastMetricUpdate: time.Now(),
		log:              prod.Log,
	}

	tgo.Metric.New(spoolingMetricWrite + streamName)
	tgo.Metric.New(spoolingMetricWriteSec + streamName)
	tgo.Metric.New(spoolingMetricRead + streamName)
	tgo.Metric.New(spoolingMetricReadSec + streamName)
	go spool.read()
	return spool
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
		base := filepath.Base(file.Name())
		number, _ := tstrings.Btoi([]byte(base)) // Because we need leading zero support
		min = tmath.MinI(min, int(number))
		max = tmath.MaxI(max, int(number))
	}
	return min, max
}

func (spool *spoolFile) openOrRotate() bool {
	fileSize := int64(0)
	if spool.file != nil {
		fileInfo, _ := spool.file.Stat()
		fileSize = fileInfo.Size()
	}

	if spool.file == nil || fileSize >= spool.prod.maxFileSize || time.Since(spool.fileCreated) > spool.prod.maxFileAge {
		_, maxSuffix := spool.getFileNumbering()

		// Reopen spooling file
		var err error
		oldFile := spool.file
		spoolFileName := fmt.Sprintf(spoolFileFormatString, spool.basePath, maxSuffix+1)
		spool.file, err = os.OpenFile(spoolFileName, os.O_WRONLY|os.O_CREATE, 0600)

		// Close current file
		if oldFile != nil {
			spool.batch.WaitForFlush(time.Duration(0))
			oldFile.Close()
		}

		spool.assembly.SetWriter(spool.file)
		spool.fileCreated = time.Now()
		if err != nil {
			spool.log.Error.Print("Spooling: ", err)
			return false // ### return, could not open file ###
		}
		spool.log.Debug.Print("Spooler opened ", spoolFileName, " for writing")
	}

	return true
}

func (spool *spoolFile) read() {
	spool.prod.AddWorker()
	defer spool.prod.WorkerDone()

	for spool.prod.IsActive() {
		minSuffix, _ := spool.getFileNumbering()

		spoolFileName := fmt.Sprintf(spoolFileFormatString, spool.basePath, minSuffix)
		if minSuffix == 0 || minSuffix > maxSpoolFileNumber || (spool.file != nil && spool.file.Name() == spoolFileName) {
			if minSuffix > maxSpoolFileNumber {
				spool.log.Debug.Print("Spool read sleeps (no file)")
			} else {
				spool.log.Debug.Printf("Spool read waits for %s", spoolFileName)
			}
			time.Sleep(spool.prod.maxFileAge / 2)
			continue // ### continue, try again ###
		}

		file, err := os.OpenFile(spoolFileName, os.O_RDONLY, 0600)
		if err != nil {
			spool.log.Error.Print("Spool read open error ", err)
			continue // ### continue, try again ###
		}

		spool.log.Debug.Print("Spooler opened ", spoolFileName, " for reading")
		reader := bufio.NewReader(file)

		for spool.prod.IsActive() {
			// Only spool back if target is not busy
			if spool.source != nil && spool.source.IsBlocked() {
				time.Sleep(time.Millisecond * 100)
				continue // ### contine, busy source ###
			}
			// Read one line (might require multiple reads)
			buffer, isPartial, err := reader.ReadLine()
			for isPartial && err == nil {
				var appendix []byte
				appendix, isPartial, err = reader.ReadLine()
				buffer = append(buffer, appendix...)
			}
			// Any error cancels the loop
			if err != nil {
				if err != io.EOF {
					spool.log.Error.Print("Spool read error: ", err)
				}
				break // ### break, read error or EOF ###
			}

			// Base64 decode, than deserialize
			decoded := make([]byte, base64.StdEncoding.DecodedLen(len(buffer)))
			if size, err := base64.StdEncoding.Decode(decoded, buffer); err != nil {
				spool.log.Error.Print("Spool file read: ", err)
			} else if msg, err := core.DeserializeMessage(decoded[:size]); err != nil {
				spool.log.Error.Print("Spool file read: ", err)
			} else {
				spool.prod.routeToOrigin(msg)
			}
		}

		// Close and remove file
		spool.log.Debug.Print("Spooler removes ", spoolFileName)
		file.Close()
		os.Remove(spoolFileName)
	}
}
