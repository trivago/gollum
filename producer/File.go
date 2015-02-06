package producer

import (
	"compress/gzip"
	"fmt"
	"github.com/trivago/gollum/shared"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	fileProducerTimestamp = "2006-01-02_15"
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
//
// Rotate if set to true the logs will rotate after reaching certain thresholds.
//
// RotateTimeoutMin defines a timeout in minutes that will cause the logs to
// rotate. Can be set in parallel with RotateSizeMB. By default this is set to
// 1440 (i.e. 1 Day).
//
// RotateAt defines specific timestamp as in "HHMM" when the log should be
// rotated. When left empty this setting is ignored. By default this setting is
// disabled.
type File struct {
	standardProducer
	file             *os.File
	batch            *shared.MessageBuffer
	fileDir          string
	fileName         string
	fileExt          string
	fileCreated      time.Time
	rotateAt         string
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
	prod.rotateSizeByte = int64(conf.GetInt("RotateSizeMB", 1024)) << 20
	prod.rotateAt = conf.GetString("RotateAt", "")
	prod.file = nil

	prod.fileDir = filepath.Dir(logFile)
	prod.fileExt = filepath.Ext(logFile)
	prod.fileName = filepath.Base(logFile)
	prod.fileName = prod.fileName[:len(prod.fileName)-len(prod.fileExt)]

	return prod, nil
}

func (prod *File) needsRotate() (bool, error) {
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

func (prod File) compressAndCloseLog(sourceFile *os.File) {
	// Generate file to zip into
	sourceFileName := sourceFile.Name()
	sourceBase := filepath.Base(sourceFileName)
	sourceBase = sourceBase[:len(sourceBase)-len(prod.fileExt)]

	targetFileName := fmt.Sprintf("%s/%s.gz", prod.fileDir, sourceBase)

	targetFile, err := os.OpenFile(targetFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		shared.Log.Error("File compress error:", err)
		sourceFile.Close()
		return
	}

	// Create zipfile
	shared.Log.Note("Compressing " + sourceFileName)
	sourceFile.Seek(0, 0)
	targetWriter := gzip.NewWriter(targetFile)
	_, err = io.Copy(targetWriter, sourceFile)

	sourceFile.Close()
	targetWriter.Close()
	targetFile.Close()

	if err != nil {
		shared.Log.Warning("Compression failed:", err)
		err = os.Remove(targetFileName)
		if err != nil {
			shared.Log.Error("Compressed file remove failed:", err)
		}
		return
	}

	// Remove original log
	err = os.Remove(sourceFileName)
	if err != nil {
		shared.Log.Error("Uncompressed file remove failed:", err)
	}
}

func (prod *File) openLog() error {
	if rotate, err := prod.needsRotate(); !rotate {
		return err
	}

	// Generate the log filename based on rotation, existing files, etc.
	var logFile string

	if !prod.rotate {
		logFile = fmt.Sprintf("%s/%s%s", prod.fileDir, prod.fileName, prod.fileExt)
	} else {
		timestamp := time.Now().Format(fileProducerTimestamp)
		signature := fmt.Sprintf("%s_%s", prod.fileName, timestamp)
		counter := 0

		files, _ := ioutil.ReadDir(prod.fileDir)
		for _, file := range files {
			if strings.Contains(file.Name(), signature) {
				counter++
			}
		}

		if counter == 0 {
			logFile = fmt.Sprintf("%s/%s%s", prod.fileDir, signature, prod.fileExt)
		} else {
			logFile = fmt.Sprintf("%s/%s_%d%s", prod.fileDir, signature, counter, prod.fileExt)
		}
	}

	// Close existing log
	if prod.file != nil {
		go prod.compressAndCloseLog(prod.file)
		prod.file = nil
		shared.Log.Note("Log rotated.")
	}

	// (Re)open logfile
	var err error
	prod.file, err = os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	// Create "current" symlink
	prod.fileCreated = time.Now()
	symLinkName := fmt.Sprintf("%s/%s_current", prod.fileDir, prod.fileName)

	os.Remove(symLinkName)
	os.Symlink(logFile, symLinkName)

	return err
}

func (prod *File) write() {
	if err := prod.openLog(); err != nil {
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
