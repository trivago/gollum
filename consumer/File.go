package consumer

import (
	"github.com/trivago/gollum/log"
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

// File consumer plugin
// Configuration example
//
// - "consumer.File":
//   Enable: true
//   File: "test.txt"
//   Offset: "Current"
//	 Delimiter: "\n"
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
	shared.ConsumerBase
	file             *os.File
	openMutex        *sync.Mutex
	fileName         string
	continueFileName string
	delimiter        string
	seek             int
	seekOffset       int64
	quit             bool
	persistSeek      bool
}

func init() {
	shared.RuntimeType.Register(File{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *File) Configure(conf shared.PluginConfig) error {
	err := cons.ConsumerBase.Configure(conf)
	if err != nil {
		return err
	}

	if !conf.HasValue("File") {
		return shared.NewConsumerError("No file configured for consumer.File")
	}

	escapeChars := strings.NewReplacer("\\n", "\n", "\\r", "\r", "\\t", "\t")

	cons.file = nil
	cons.openMutex = new(sync.Mutex)
	cons.fileName = conf.GetString("File", "")
	cons.delimiter = escapeChars.Replace(conf.GetString("Delimiter", "\n"))
	cons.quit = false
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

		cons.reopen()
	}

	return nil
}

func (cons *File) postAndPersist(data []byte) {
	cons.PostMessageFromSlice(data)
	cons.seekOffset, _ = cons.file.Seek(0, 1)
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

func (cons *File) reopen() {
	if !cons.persistSeek {
		return
	}

	cons.openMutex.Lock()
	defer cons.openMutex.Unlock()

	if cons.file != nil {
		fileHandle := cons.file
		cons.file = nil
		fileHandle.Close()
	}

	baseFileName := cons.realFileName()
	pathDelimiter := strings.NewReplacer("/", "_", ".", "_")
	cons.continueFileName = "/tmp/gollum" + pathDelimiter.Replace(baseFileName) + ".idx"

	fileContents, err := ioutil.ReadFile(cons.continueFileName)
	if err == nil {
		cons.seekOffset, err = strconv.ParseInt(string(fileContents), 10, 64)
	}
	if err != nil {
		cons.seekOffset = 0
	}
}

func (cons *File) readFrom(threads *sync.WaitGroup) {
	var buffer shared.BufferedReader
	var err error

	if cons.persistSeek {
		buffer = shared.CreateBufferedReader(fileBufferGrowSize, cons.postAndPersist)
	} else {
		buffer = shared.CreateBufferedReader(fileBufferGrowSize, cons.PostMessageFromSlice)
	}

	printFileOpenError := true

	for !cons.quit {
		if cons.file == nil {
			cons.openMutex.Lock()
			cons.file, err = os.OpenFile(cons.realFileName(), os.O_RDONLY, 0666)

			if err != nil {
				cons.openMutex.Unlock()
				if printFileOpenError {
					Log.Error.Print("File open error - ", err)
					printFileOpenError = false
				}
				time.Sleep(3 * time.Second)
			} else {
				cons.seekOffset, _ = cons.file.Seek(cons.seekOffset, cons.seek)
				printFileOpenError = true
				cons.openMutex.Unlock()
			}
		}

		if cons.file != nil {
			err = buffer.Read(cons.file, cons.delimiter)

			if err != nil && cons.file != nil && !cons.quit {
				if err == io.EOF {
					runtime.Gosched()
				} else {
					Log.Error.Print("Error reading file - ", err)
					cons.file.Close()
					cons.file = nil
				}
			}
		}
	}

	cons.MarkAsDone()
}

// Consume listens to stdin.
func (cons File) Consume(threads *sync.WaitGroup) {
	cons.quit = false

	go cons.readFrom(threads)

	defer func() {
		cons.quit = true
		cons.file.Close()
	}()

	cons.DefaultControlLoop(threads, cons.reopen)
}
