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

package consumer

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"github.com/trivago/tgo/tio"
	"github.com/trivago/tgo/tsync"
)

type observableFile struct {
	handle         *os.File
	fileName       string
	offsetFileName string
	cursor         fileCursor
	buffer         *tio.BufferedReader
	stopIfNotExist bool

	lastStatCheck time.Time
	retryDelay    time.Duration
	pollDelay     time.Duration
	log           logrus.FieldLogger
}

type fileCursor struct {
	whence int
	offset int64
}

func (fs *observableFile) close() error {
	if fs.handle != nil {
		return fs.handle.Close()
	}
	return nil
}

func (fs *observableFile) getActualFilename() string {
	actualFileName := fs.fileName
	if evalFileName, err := filepath.EvalSymlinks(actualFileName); err == nil {
		actualFileName = evalFileName
	}
	if absFileName, err := filepath.Abs(actualFileName); err == nil {
		actualFileName = absFileName
	}
	return actualFileName
}

func (fs *observableFile) hasRotated(currentName string) bool {
	if fs.handle == nil {
		return false
	}

	actualFileName := fs.getActualFilename()
	if fs.handle.Name() != actualFileName {
		return true
	}

	if time.Since(fs.lastStatCheck) < fs.pollDelay {
		return false
	}

	newStat, newStatErr := os.Stat(actualFileName)
	oldStat, oldStatErr := fs.handle.Stat()
	fs.lastStatCheck = time.Now()
	return newStatErr != oldStatErr || !os.SameFile(newStat, oldStat)
}

func (fs *observableFile) storeOffset() {
	fs.cursor.offset, _ = fs.handle.Seek(0, io.SeekCurrent)
	fs.saveOffset(fs.cursor.offset)
}

func (fs *observableFile) saveOffset(offset int64) {
	if len(fs.offsetFileName) == 0 {
		return
	}
	offsetAsString := strconv.FormatInt(offset, 10)
	if err := ioutil.WriteFile(fs.offsetFileName, []byte(offsetAsString), 0644); err != nil {
		fs.log.WithError(err).Error("Failed to store offset")
	}
}

func (fs *observableFile) scrape(fileName string, enqueue func([]byte), onRotate func()) {
	// Try to open the current file
	if fs.handle == nil {
		handle, err := os.OpenFile(fileName, os.O_RDONLY, 0444)
		if err != nil {
			fs.log.Warning("Failed to open file")
			time.Sleep(fs.retryDelay)
			return // wait between retries
		}

		if fs.cursor.offset, err = handle.Seek(fs.cursor.offset, fs.cursor.whence); err != nil {
			fs.log.WithError(err).Warning("Failed to seek to given offset")
		}
		fs.handle = handle
	}

	// Try to scrape the file
	err := fs.buffer.ReadAll(fs.handle, enqueue)

	switch err {
	case nil:
	case io.EOF:
		if fs.hasRotated(fileName) {
			fs.log.Info("File rotated")
			fs.handle.Close()
			fs.handle = nil
			fs.buffer.Reset(0)

			fs.cursor.whence = io.SeekStart
			fs.cursor.offset = 0
			fs.saveOffset(fs.cursor.offset)
			fs.log.Info("Offset file reseted")
			onRotate()
		}
	default:
		fs.log.WithError(err).Error("Failed to read file")
		fs.handle.Close()
		fs.handle = nil
		fs.buffer.Reset(0)
	}
}

func (fs *observableFile) observePoll(enqueue func([]byte), done <-chan struct{}) {
	spin := tsync.NewCustomSpinner(fs.pollDelay)
	actualFileName := fs.getActualFilename()
	logger := fs.log

	for {
		select {
		default:
		case <-done:
			return // exit requested
		}

		if fs.handle == nil {
			actualFileName = fs.getActualFilename()
			fs.log = logger.WithFields(logrus.Fields{
				"Scraping": actualFileName,
			})

			if _, err := os.Stat(actualFileName); err != nil {
				fs.log.WithError(err).Warning("Failed to get file stat")
				if fs.stopIfNotExist && os.IsNotExist(err) {
					return
				}
				time.Sleep(fs.retryDelay)
				continue // retry
			}
		}

		fs.scrape(actualFileName, enqueue, spin.Reset)
		spin.Yield()
	}
}

func (fs *observableFile) observeFSNotify(enqueue func([]byte), done <-chan struct{}) {
	notify, err := fsnotify.NewWatcher()
	if err != nil {
		fs.log.WithError(err).Error("Failed to start fsnotify watcher")
		return
	}
	defer notify.Close()
	logger := fs.log

	for {
		select {
		default:
		case <-done:
			return // exit requested
		}

		// Try to attach file to fsnotify
		actualFileName := fs.getActualFilename()
		fs.log = logger.WithFields(logrus.Fields{
			"Scraping": actualFileName,
		})

		if _, err := os.Stat(actualFileName); err != nil {
			fs.log.WithError(err).Warning("Failed to get file stat")
			if fs.stopIfNotExist && os.IsNotExist(err) {
				return
			}
			time.Sleep(fs.retryDelay)
			continue // retry
		}

		if err := notify.Add(actualFileName); err != nil {
			fs.log.WithError(err).Error("Error adding fsnotify watcher")
			time.Sleep(fs.retryDelay)
			continue // retry
		}

		rotated := false
		for !rotated {
			select {
			case event := <-notify.Events:
				containsMoveEvent := event.Op&fsnotify.Rename != 0 || event.Op&fsnotify.Remove != 0
				containsWriteEvent := event.Op&fsnotify.Write == 0

				switch {
				case containsMoveEvent:
					fs.log.Debugf("Detected file move event %s", event.Name)
					fs.scrape(actualFileName, enqueue, func() {})
					rotated = true
					notify.Remove(actualFileName)

				case containsWriteEvent:
					fs.log.Debugf("Detected file modification event %s", event.Name)
					fs.scrape(actualFileName, enqueue, func() {
						rotated = true
						notify.Remove(actualFileName)
					})
				}

			case err := <-notify.Errors:
				fs.log.WithError(err).Error("Fsnotify reported an error")
				rotated = true
				notify.Remove(actualFileName)

			case <-done:
				return //  exit requested
			}
		}
	}
}
