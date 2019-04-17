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
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo"
	"github.com/trivago/tgo/tio"
)

const (
	fileBufferGrowSize = 1024
	fileOffsetStart    = "oldest"
	fileOffsetEnd      = "newest"
)

const (
	observeModePoll  = "poll"
	observeModeWatch = "watch"
)

const (
	stopIfNotExist  = true
	retryIfNotExist = false
)

// File consumer plugin
//
// The File consumer reads messages from a file, looking for a customizable
// delimiter sequence that marks the end of a message. If the file is part of
// e.g. a log rotation, the consumer can be set to read from a symbolic link
// pointing to the current file and (optionally) be told to reopen the file
// by sending a SIGHUP. A symlink to a file will automatically be reopened
// if the underlying file is changed.
//
// Metadata
//
// *NOTE: The metadata will only set if the parameter `SetMetadata` is active.*
//
// - file: The file name of the consumed file (set)
//
// - dir: The directory of the consumed file (set)
//
// Parameters
//
// - File: This value is a mandatory setting and contains the name of the
// file to read. This field supports glob patterns.
// If the file pointed to is a symlink, changes to the symlink will be
// detected. The file will be watched for changes, so active logfiles can
// be scraped, too.
//
// - OffsetFilePath: This value defines a path where the individual, current
// file offsets are stored. The filename will the name and extension of the
// source file plus the extension ".offset". If the consumer is restarted,
// these offset files are used to continue reading from the previous position.
// To disable this setting, set it to "".
// By default this parameter is set to "".
//
// - Delimiter: This value defines the delimiter sequence to expect at the
// end of each message in the file.
// By default this parameter is set to "\n".
//
// - ObserveMode: This value select how the source file is observed. Available
// values are `poll` and `watch`.  NOTE: The watch implementation uses
// the [fsnotify/fsnotify](https://github.com/fsnotify/fsnotify) package.
// If your source file is rotated (moved or removed), please verify that
// your file system and distribution support the `RENAME` and `REMOVE` events;
// the consumer's stability depends on them.
// By default this parameter is set to `poll`.
//
// - DefaultOffset: This value defines the default offset from which to start
// reading within the file. Valid values are  "oldest" and "newest". If OffsetFile
// is defined and the file exists, the DefaultOffset parameter is ignored.
// By default this parameter is set to "newest".
//
// - PollingDelayMs: This value defines the duration in milliseconds the consumer
// waits between checking the source file for new content after hitting the
// end of file (EOF). NOTE: This settings only takes effect if the consumer is
// running in `poll` mode!
// By default this parameter is set to "100".
//
// - RetryDelaySec: This value defines the duration in seconds the consumer waits
// between retries, e.g. after not being able to open a file.
// By default this parameter is set to "3".
//
// - DirScanIntervalSec: Only applies when using globs. This setting will define the
// interval in secnds in which the glob will be re-evaluated and new files can be
// scraped. By default this parameter is set to "10".
//
// - SetMetadata: When this value is set to "true", the fields mentioned in the metadata
// section will be added to each message. Adding metadata will have a
// performance impact on systems with high throughput.
// By default this parameter is set to "false".
//
// - BlackList: A regular expression matching file paths to NOT read. When both
// BlackList and WhiteList are defined, the WhiteList takes precedence.
// This setting is only used when glob expressions (*, ?) are present in the
// filename. The path checked is the one before symlink evaluation.
// By default this parameter is set to "".
//
// - WhiteList: A regular expression matching file paths to read. When both
// BlackList and WhiteList are defined, the WhiteList takes precedence.
// This setting is only used when glob expressions (*, ?) are present in the
// filename. The path checked is the one before symlink evaluation.
// By default this parameter is set to "".
//
// Examples
//
// This example will read all the `.log` files `/var/log/` into one stream and
// create a message for each new entry. If the file starts with `sys` it is ignored
//
//  FileIn:
//    Type: consumer.File
//    File: /var/log/*.log
//    BlackList '^sys.*'
//    DefaultOffset: newest
//    OffsetFilePath: ""
//    Delimiter: "\n"
//    ObserveMode: poll
//    PollingDelay: 100
//
type File struct {
	core.SimpleConsumer `gollumdoc:"embed_type"`

	fileName         string        `config:"Files" default:"/var/log/*.log"`
	offsetFilePath   string        `config:"OffsetFilePath"`
	pollingDelay     time.Duration `config:"PollingDelayMs" default:"100" metric:"ms"`
	retryDelay       time.Duration `config:"RetryDelaySec" default:"3" metric:"s"`
	dirScanInterval  time.Duration `config:"DirScanIntervalSec" default:"10" metric:"s"`
	delimiter        string        `config:"Delimiter" default:"\n"`
	observeMode      string        `config:"ObserveMode" default:"poll"`
	hasToSetMetadata bool          `config:"SetMetadata" default:"false"`
	defaultOffset    string        `config:"DefaultOffset" default:"newest"`
	blackListString  string        `config:"BlackList"`
	whiteListString  string        `config:"WhiteList"`

	observedFiles *sync.Map
	done          chan struct{}
	isBlackListed func(string) bool
}

func init() {
	core.TypeRegistry.Register(File{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *File) Configure(conf core.PluginConfigReader) {
	cons.done = make(chan struct{})
	cons.observedFiles = new(sync.Map)

	// TODO: support manual roll again
	//cons.SetRollCallback(cons.onRoll)
	cons.SetStopCallback(func() {
		close(cons.done)
	})

	// restore default observer mode for invalid config settings
	if cons.observeMode != observeModePoll && cons.observeMode != observeModeWatch {
		cons.Logger.Warningf("Unknown observe mode '%s'. Using poll", cons.observeMode)
		cons.observeMode = observeModePoll
	}

	cons.configureBlacklist(conf)
}

func (cons *File) configureBlacklist(conf core.PluginConfigReader) {
	var (
		err       error
		blackList *regexp.Regexp
		whiteList *regexp.Regexp
	)

	if len(cons.blackListString) > 0 {
		blackList, err = regexp.Compile(cons.blackListString)
		conf.Errors.Push(err)
	}
	if len(cons.whiteListString) > 0 {
		whiteList, err = regexp.Compile(cons.whiteListString)
		conf.Errors.Push(err)
	}

	// Define how to do blacklisting based on the values given above
	switch {
	case blackList == nil && whiteList == nil:
		cons.isBlackListed = func(string) bool { return false }

	case blackList == nil:
		cons.isBlackListed = func(filename string) bool { return !whiteList.MatchString(filename) }

	case whiteList == nil:
		cons.isBlackListed = func(filename string) bool { return blackList.MatchString(filename) }

	default:
		cons.isBlackListed = func(filename string) bool {
			return blackList.MatchString(filename) && !whiteList.MatchString(filename)
		}
	}
}

func (cons *File) newObservedFile(name string, stopIfNotExist bool) *observableFile {
	logger := cons.Logger.WithFields(logrus.Fields{
		"File": name,
	})

	offsetFileName := ""
	defaultOffset := strings.ToLower(cons.defaultOffset)
	cursor := fileCursor{whence: io.SeekStart}

	switch {
	case cons.offsetFilePath != "":
		offsetFileName = fmt.Sprintf("%s/%s.offset", cons.offsetFilePath, filepath.Base(cons.fileName))
		if offsetFileData, err := ioutil.ReadFile(offsetFileName); err != nil {
			logger.WithError(err).Errorf("Failed to open offset file %s", offsetFileName)
		} else {
			if offset, err := strconv.ParseInt(string(offsetFileData), 10, 64); err != nil {
				logger.WithError(err).Errorf("Error reading offset number from %s", offsetFileName)
			} else {
				cursor.offset = offset
			}
		}

	case defaultOffset == fileOffsetEnd:
		cursor.whence = io.SeekEnd

	case defaultOffset == fileOffsetStart:
	default:
	}

	logger.Info("Starting file scraper")

	return &observableFile{
		fileName:       name,
		offsetFileName: offsetFileName,
		cursor:         cursor,
		stopIfNotExist: stopIfNotExist,
		retryDelay:     cons.retryDelay,
		pollDelay:      cons.pollingDelay,
		buffer:         tio.NewBufferedReader(fileBufferGrowSize, tio.BufferedReaderFlagDelimiter, 0, cons.delimiter),
		log:            logger,
	}
}

func (cons *File) observeFile(name string, stopIfNotExist bool) {
	defer cons.WorkerDone()

	file := cons.newObservedFile(name, stopIfNotExist)
	defer file.close()

	cons.observedFiles.Store(name, file)
	defer cons.observedFiles.Delete(name)

	enqueue := cons.Enqueue

	if cons.hasToSetMetadata {
		dirName, fileName := filepath.Split(name)
		enqueue = func(data []byte) {
			metaData := core.NewMetadata()
			metaData.Set("file", fileName)
			metaData.Set("dir", dirName)
			cons.EnqueueWithMetadata(data, metaData)
		}
	}

	if cons.offsetFilePath != "" {
		enqueue = func(data []byte) {
			cons.Enqueue(data)
			file.storeOffset()
		}
	}

	switch cons.observeMode {
	case observeModeWatch:
		file.observeFSNotify(enqueue, cons.done)
	default:
		file.observePoll(enqueue, cons.done)
	}
}

func (cons *File) observeFiles() {
	defer cons.WorkerDone()

	if !strings.ContainsAny(cons.fileName, "*?") {
		cons.observeFile(cons.fileName, retryIfNotExist) // blocking
		return
	}

	// Glob needs to be re-evaluated to find new files.
	for {
		fileNames, err := filepath.Glob(cons.fileName)
		if err != nil {
			cons.Logger.Warningf("Failed to evaluate glob '%s'", cons.fileName)
			return
		}

		cons.Logger.Debugf("Evaluating glob returned %d files to scrape", len(fileNames))
		for i := range fileNames {
			if cons.isBlackListed(fileNames[i]) {
				continue
			}

			if _, ok := cons.observedFiles.Load(fileNames[i]); !ok {
				cons.AddWorker()
				go cons.observeFile(fileNames[i], stopIfNotExist)
			}
		}

		select {
		case <-time.After(cons.dirScanInterval):
		case <-cons.done:
			return
		}
	}
}

// Consume opens the given file(s) for reading
func (cons *File) Consume(workers *sync.WaitGroup) {
	go tgo.WithRecoverShutdown(func() {
		cons.AddMainWorker(workers)
		cons.observeFiles()
	})

	cons.ControlLoop()
}
