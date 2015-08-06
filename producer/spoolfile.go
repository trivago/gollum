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
	"fmt"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

type spoolFile struct {
	file        *os.File
	batch       core.MessageBatch
	assembly    core.WriterAssembly
	fileCreated time.Time
	streamName  string
	prod        *Spooling
	source      core.MessageSource
}

const maxSpoolFileNumber = 99999999 // maximum file number defined by %08d -> 8 digits
const spoolFileFormatString = "%s/%08d.spl"

func newSpoolFile(prod *Spooling, streamName string, source core.MessageSource) *spoolFile {
	spool := &spoolFile{
		file:        nil,
		batch:       core.NewMessageBatch(prod.batchMaxCount),
		assembly:    core.NewWriterAssembly(nil, prod.Drop, prod.GetFormatter()),
		fileCreated: time.Now(),
		streamName:  streamName,
		prod:        prod,
		source:      source,
	}

	shared.Metric.New(spoolingMetricName + streamName)
	shared.Metric.New(spooledMetricName + streamName)
	go spool.read()
	return spool
}

func (spool *spoolFile) flush() {
	spool.batch.Flush(spool.assembly.Write)
}

func (spool *spoolFile) close() {
	spool.batch.Flush(spool.assembly.Flush)
	spool.batch.WaitForFlush(spool.prod.GetShutdownTimeout())
	spool.file.Close()
}

func (spool *spoolFile) getFileNumbering(path string) (min int, max int) {
	min, max = maxSpoolFileNumber+1, 0
	files, _ := ioutil.ReadDir(path)
	for _, file := range files {
		base := filepath.Base(file.Name())
		number, _ := shared.Btoi([]byte(base)) // Because we need leading zero support
		min = shared.MinI(min, int(number))
		max = shared.MaxI(max, int(number))
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
		path := spool.prod.path + "/" + spool.streamName
		_, maxSuffix := spool.getFileNumbering(path)

		// Reopen spooling file
		var err error
		oldFile := spool.file
		spoolFileName := fmt.Sprintf(spoolFileFormatString, path, maxSuffix+1)
		spool.file, err = os.OpenFile(spoolFileName, os.O_WRONLY|os.O_CREATE, 0600)

		// Close current file
		if oldFile != nil {
			spool.batch.WaitForFlush(time.Duration(0))
			oldFile.Close()
		}

		spool.assembly.SetWriter(spool.file)
		spool.fileCreated = time.Now()
		if err != nil {
			Log.Error.Print("Spooling: ", err)
			return false // ### return, could not open file ###
		}
		Log.Debug.Print("Spooler opened ", spoolFileName, " for writing")
	}

	return true
}

func (spool *spoolFile) read() {
	spool.prod.AddWorker()
	defer spool.prod.WorkerDone()

	path := spool.prod.path + "/" + spool.streamName

	for spool.prod.IsActive() {
		minSuffix, _ := spool.getFileNumbering(path)

		spoolFileName := fmt.Sprintf(spoolFileFormatString, path, minSuffix)
		if minSuffix == 0 || minSuffix > maxSpoolFileNumber || (spool.file != nil && spool.file.Name() == spoolFileName) {
			if minSuffix > maxSpoolFileNumber {
				Log.Debug.Print("Spool read sleeps (no file)")
			} else {
				Log.Debug.Printf("Spool read waits for %s", spoolFileName)
			}
			time.Sleep(spool.prod.maxFileAge / 2)
			continue // ### continue, try again ###
		}

		file, err := os.OpenFile(spoolFileName, os.O_RDONLY, 0600)
		if err != nil {
			Log.Error.Print("Spool read open error ", err)
			continue // ### continue, try again ###
		}

		Log.Debug.Print("Spooler opened ", spoolFileName, " for reading")
		reader := bufio.NewReader(file)
		for {
			// Only spool back if target is not busy
			if spool.source != nil && spool.source.IsBlocked() {
				time.Sleep(time.Second)
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
					Log.Error.Print("Spool read error: ", err)
				}
				break // ### break, read error or EOF ###
			}
			// Deserialize from string
			msg, err := core.DeserializeMessage(string(buffer))
			if err != nil {
				Log.Error.Print("Spool file read: ", err)
			} else {
				spool.prod.routeToOrigin(msg)
			}
		}

		// Close and remove file
		Log.Debug.Print("Spooler removes ", spoolFileName)
		file.Close()
		os.Remove(spoolFileName)
	}
}
