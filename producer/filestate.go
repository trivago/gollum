// Copyright 2015-2017 trivago GmbH
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
	"errors"
	"github.com/sirupsen/logrus"
	"github.com/trivago/gollum/core"
	"io"
	"strconv"
	"strings"
	"time"
)

// BatchedWriterAssembly is a helper struct for io.Writer compatible classes that use batch directly for resources
type BatchedWriterAssembly struct {
	writer       BatchedWriter
	batch        core.MessageBatch
	assembly     core.WriterAssembly
	created      time.Time
	flushTimeout time.Duration
	logger       logrus.FieldLogger

	batchTimeout    time.Duration
	batchFlushCount int
}

// BatchedWriter is an interface for different file writer like disk, s3, etc.
type BatchedWriter interface {
	io.WriteCloser
	Name() string // base name of the file/resource
	Size() int64  // length in bytes for regular files; system-dependent for others
	//Created() time.Time
	IsAccessible() bool
}

// NewBatchedWriterAssembly returns a new BatchedWriterAssembly instance
func NewBatchedWriterAssembly(batchMaxCount int, batchTimeout time.Duration, batchFlushCount int, modulator core.Modulator, tryFallback func(*core.Message),
	timeout time.Duration, logger logrus.FieldLogger) *BatchedWriterAssembly {
	return &BatchedWriterAssembly{
		batch:           core.NewMessageBatch(batchMaxCount),
		assembly:        core.NewWriterAssembly(nil, tryFallback, modulator),
		flushTimeout:    timeout,
		batchTimeout:    batchTimeout,
		batchFlushCount: batchFlushCount,
		logger:          logger,
	}
}

// Flush flush the batch
func (bwa *BatchedWriterAssembly) Flush() {
	if bwa.writer != nil {
		bwa.assembly.SetWriter(bwa.writer)
		bwa.batch.Flush(bwa.assembly.Write)
	} else {
		bwa.batch.Flush(bwa.assembly.Flush)
	}
}

// Close closes batch and writer
func (bwa *BatchedWriterAssembly) Close() {
	if bwa.writer != nil {
		bwa.assembly.SetWriter(bwa.writer)
		bwa.batch.Close(bwa.assembly.Write, bwa.flushTimeout)
	} else {
		bwa.batch.Close(bwa.assembly.Flush, bwa.flushTimeout)
	}
	bwa.writer.Close()
}

// FlushOnTimeOut checks if timeout or slush count reached and flush in this case
func (bwa *BatchedWriterAssembly) FlushOnTimeOut() {
	if bwa.batch.ReachedTimeThreshold(bwa.batchTimeout) || bwa.batch.ReachedSizeThreshold(bwa.batchFlushCount) {
		bwa.Flush()
	}
}

// NeedsRotate evaluate if the BatchedWriterAssembly need to rotate by the FileRotateConfig
func (bwa *BatchedWriterAssembly) NeedsRotate(rotate FileRotateConfig, forceRotate bool) (bool, error) {
	// File does not exist?
	if bwa.writer == nil {
		return true, nil
	}

	// File can be accessed?
	if bwa.writer.IsAccessible() == false {
		return false, errors.New("Can' access file to rotate")
	}

	// File needs rotation?
	if !rotate.enabled {
		return false, nil
	}

	if forceRotate {
		return true, nil
	}

	// File is too large?
	if bwa.writer.Size() >= rotate.sizeByte {
		return true, nil // ### return, too large ###
	}

	// File is too old?
	if time.Since(bwa.created) >= rotate.timeout {
		return true, nil // ### return, too old ###
	}

	// RotateAt crossed?
	if rotate.atHour > -1 && rotate.atMinute > -1 {
		now := time.Now()
		rotateAt := time.Date(now.Year(), now.Month(), now.Day(), rotate.atHour, rotate.atMinute, 0, 0, now.Location())

		if bwa.created.Sub(rotateAt).Minutes() < 0 {
			return true, nil // ### return, too old ###
		}
	}

	// nope, everything is ok
	return false, nil
}

// -- FileRotateConfig --

// FileRotateConfig defines the rotation settings
type FileRotateConfig struct {
	timeout   time.Duration `config:"Rotation/TimeoutMin" default:"1440" metric:"min"`
	sizeByte  int64         `config:"Rotation/SizeMB" default:"1024" metric:"mb"`
	zeroPad   int           `config:"Rotation/ZeroPadding" default:"0"`
	enabled   bool          `config:"Rotation/Enable" default:"false"`
	compress  bool          `config:"Rotation/Compress" default:"false"`
	timestamp string        `config:"Rotation/Timestamp" default:"2006-01-02_15"`
	atHour    int           `config:"Rotation/AtHour" default:"-1"`
	atMinute  int           `config:"Rotation/AtMin" default:"-1"`
}

// Configure method for interface implementation
func (rotate *FileRotateConfig) Configure(conf core.PluginConfigReader) {
	rotateAt := conf.GetString("Rotation/At", "")
	if rotateAt != "" {
		parts := strings.Split(rotateAt, ":")
		rotateAtHour, _ := strconv.ParseInt(parts[0], 10, 8)
		rotateAtMin, _ := strconv.ParseInt(parts[1], 10, 8)

		rotate.atHour = int(rotateAtHour)
		rotate.atMinute = int(rotateAtMin)
	}
}
