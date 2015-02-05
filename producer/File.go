package producer

import (
	"fmt"
	"github.com/trivago/gollum/shared"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	fileProducerTimestamp = "2006-01-02-150405.000"
)

// File producer plugin
// Configuration example
//
// - "producer.File":
//   Enable: true
//   File: "/var/log/gollum.log"
//   BatchSize: 4096
//   BatchSizeThreshold: 16777216
//   BatchTimeoutSec: 2
//   Rotate: false
//   RotateTimeoutMin: 1440
//   RotateSizeMB: 1024
//
// File contains the path to the log file to write.
// By default this is set to /var/prod/gollum.log.
//
// BatchSize defines the number of bytes to be buffered before they are written
// to disk. By default this is set to 8KB.
//
// BatchSizeThreshold defines the maximum number of bytes to buffer before
// messages get dropped. Any message that crosses the threshold is dropped.
// By default this is set to 8MB.
//
// BatchTimeoutSec defines the maximum number of seconds to wait after the last
// message arrived before a batch is flushed automatically. By default this is
// set to 5.
type File struct {
	standardProducer
	file             *os.File
	batch            *shared.MessageBuffer
	fileDir          string
	fileName         string
	fileExt          string
	fileCreated      time.Time
	rotateSizeByte   int64
	batchSize        int
	batchTimeoutSec  int
	rotateTimeoutMin int
	rotate           bool
}

func init() {
	shared.Plugin.Register(File{})
}

// Create creates a new producer based on the current file producer.
func (prod File) Create(conf shared.PluginConfig) (shared.Producer, error) {
	err := prod.configureStandardProducer(conf)
	if err != nil {
		return nil, err
	}

	logFile := conf.GetString("File", "/var/prod/gollum.log")
	batchSizeThreshold := conf.GetInt("BatchThreshold", 8388608)

	prod.batchSize = conf.GetInt("BatchSize", 8192)
	prod.batchTimeoutSec = conf.GetInt("BatchTimeoutSec", 5)
	prod.batch = shared.CreateMessageBuffer(batchSizeThreshold, prod.format)

	prod.rotate = conf.GetBool("Rotate", false)
	prod.rotateTimeoutMin = conf.GetInt("RotateTimeoutMin", 1440)
	prod.rotateSizeByte = int64(conf.GetInt("RotateSizeMB", 128)) << 20
	prod.file = nil

	prod.fileDir = filepath.Dir(logFile)
	prod.fileExt = filepath.Ext(logFile)
	prod.fileName = filepath.Base(logFile)
	prod.fileName = prod.fileName[:len(prod.fileName)-len(prod.fileExt)]

	return prod, nil
}

func (prod *File) needsRefresh() (bool, error) {
	// File does not exist?
	if prod.file == nil {
		return true, nil
	}

	// File needs rotation?
	if !prod.rotate {
		return false, nil
	}

	// File can be accessed?
	stats, err := prod.file.Stat()
	if err != nil {
		return false, err
	}

	// File is too large?
	if stats.Size() >= prod.rotateSizeByte {
		return true, nil // ### return, too large ###
	}

	// File is too old?
	if time.Since(prod.fileCreated).Minutes() >= float64(prod.rotateTimeoutMin) {
		return true, nil // ### return, too old ###
	}

	// nope, everything is ok
	return false, nil
}

func (prod *File) openLog() error {
	refresh, err := prod.needsRefresh()
	if !refresh {
		return err
	}

	var logFile string
	if prod.rotate {
		logFile = fmt.Sprintf("%s/%s_%s%s", prod.fileDir, prod.fileName, time.Now().Format(fileProducerTimestamp), prod.fileExt)
	} else {
		logFile = fmt.Sprintf("%s/%s%s", prod.fileDir, prod.fileName, prod.fileExt)
	}

	if prod.file != nil {
		prod.file.Close()
		prod.file = nil
	}

	prod.fileCreated = time.Now()
	prod.file, err = os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	return err
}

func (prod *File) write() {
	err := prod.openLog()
	if err != nil {
		shared.Log.Error("File rotate error:", err)
		return
	}

	prod.batch.Flush(prod.file, func(err error) {
		shared.Log.Error("File write error:", err)
	})
}

func (prod *File) writeMessage(message shared.Message) {
	prod.batch.AppendAndRelease(message)
	if prod.batch.ReachedSizeThreshold(prod.batchSize) {
		prod.write()
	}
}

func (prod File) flush() {
	for {
		select {
		case message := <-prod.messages:
			prod.writeMessage(message)
		default:
			prod.write()
			prod.batch.WaitForFlush()
			return
		}
	}
}

// Produce writes to a buffer that is dumped to a file.
func (prod File) Produce(threads *sync.WaitGroup) {
	threads.Add(1)

	defer func() {
		prod.flush()
		prod.file.Close()
		threads.Done()
	}()

	flushTicker := time.NewTicker(time.Duration(prod.batchTimeoutSec) * time.Second)

	for {
		select {
		case message := <-prod.messages:
			prod.writeMessage(message)

		case command := <-prod.control:
			if command == shared.ProducerControlStop {
				return // ### return, done ###
			}

		case <-flushTicker.C:
			if prod.batch.ReachedTimeThreshold(prod.batchTimeoutSec) {
				prod.write()
			}
		}
	}
}
