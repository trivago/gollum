// Copyright 2015-2016 mozilla
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

// +build linux,cgo,!unit

package native

import (
	"io/ioutil"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-systemd/sdjournal"
	"github.com/trivago/gollum/core"
)

// SystemdConsumer consumer plugin
//
// NOTICE: This producer is not included in standard builds. To enable it
// you need to trigger a custom build with native plugins enabled.
// The systemd consumer allows to read from the systemd journal.
//
// Parameters
//
// - SystemdUnit: This value defines what journal will be followed. This uses
// journal.add_match with _SYSTEMD_UNIT. If this value is set to "",  the filter is disabled.
// By default this parameter is set to "".
//
// - DefaultOffset: This value defines where to start reading the file. Valid values are
// "oldest" and "newest". If OffsetFile is defined the DefaultOffset setting
// will be ignored unless the file does not exist.
// By default this parameter is set to "newest".
//
// - OffsetFile: This value defines the path to a file that stores the current offset. If
// the consumer is restarted that offset is used to continue reading. Set this value to ""
// which disables the offset file.
// By default this parameter is set to "".
//
// Examples
//
// This example set up a basic systemd consumer:
//
//  exampleConsumer:
//    Type: native.Systemd
//    Streams: "*"
//    SystemdUnit: sshd.service
//
type SystemdConsumer struct {
	core.SimpleConsumer `gollumdoc:"embed_type"`
	offsetFile          string `config:"OffsetFile" default:""`
	journal             *sdjournal.Journal
}

const (
	sdOffsetTail = "newest"
	sdOffsetHead = "oldest"
)

func init() {
	core.TypeRegistry.Register(SystemdConsumer{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *SystemdConsumer) Configure(conf core.PluginConfigReader) error {
	var err error
	cons.journal, err = sdjournal.NewJournal()
	if err != nil {
		return err
	}

	if sdUnit := conf.GetString("SystemdUnit", ""); sdUnit != "" {
		err = cons.journal.AddMatch("_SYSTEMD_UNIT=" + sdUnit)
		if err != nil {
			return err
		}
	}

	// Offset
	offsetValue := strings.ToLower(conf.GetString("DefaultOffset", sdOffsetTail))

	if cons.offsetFile != "" {
		fileContents, err := ioutil.ReadFile(cons.offsetFile)
		if err != nil {
			cons.Logger.WithError(err).Error("Error reading offset file")
		}
		if string(fileContents) != "" {
			offsetValue = string(fileContents)
		}
	}

	switch offsetValue {
	case sdOffsetHead:
		if err = cons.journal.SeekHead(); err != nil {
			return err
		}

	case sdOffsetTail:
		if err = cons.journal.SeekTail(); err != nil {
			return err
		}
		// start *after* the newest record
		if _, err = cons.journal.Next(); err != nil {
			return err
		}

	default:
		offset, err := strconv.ParseUint(offsetValue, 10, 64)
		if err != nil {
			return err
		}
		if err = cons.journal.SeekRealtimeUsec(offset); err != nil {
			return err
		}
		// start *after* the specified time
		if _, err = cons.journal.Next(); err != nil {
			return err
		}
	}

	// Register close to the control message handler
	cons.SetStopCallback(cons.close)
	return conf.Errors.OrNil()
}

func (cons *SystemdConsumer) storeOffset(offset uint64) {
	ioutil.WriteFile(cons.offsetFile, []byte(strconv.FormatUint(offset, 10)), 0644)
}

func (cons *SystemdConsumer) enqueueAndPersist(data []byte, sequence uint64) {
	cons.Enqueue(data)
	cons.storeOffset(sequence)
}

func (cons *SystemdConsumer) close() {
	if cons.journal != nil {
		cons.journal.Close()
	}
	cons.WorkerDone()
}

func (cons *SystemdConsumer) read() {
	for cons.IsActive() {

		c, err := cons.journal.Next()
		if err != nil {
			cons.Logger.WithError(err).Error("Failed to advance journal")
		} else if c == 0 {
			// reached end of log
			cons.journal.Wait(1 * time.Second)
		} else {
			msg, err := cons.journal.GetDataValue("MESSAGE")
			if err != nil {
				cons.Logger.WithError(err).Error("Failed to read journal message")
			} else {
				offset, err := cons.journal.GetRealtimeUsec()
				if err != nil {
					cons.Logger.WithError(err).Error("Failed to read journal realtime")
				} else if cons.offsetFile != "" {
					cons.enqueueAndPersist([]byte(msg), offset)
				} else {
					cons.Enqueue([]byte(msg))
				}
			}
		}
	}
}

// Consume enables systemd forwarding as configured.
func (cons *SystemdConsumer) Consume(workers *sync.WaitGroup) {
	cons.AddMainWorker(workers)
	go cons.read()
	cons.ControlLoop()
}
