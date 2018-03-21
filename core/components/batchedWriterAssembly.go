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

package components

import (
	"errors"
	"io"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/tmath"
)

// BatchedWriterConfig defines batch configurations
//
// Parameters
//
// - Batch/TimeoutSec: This value defines the maximum number of seconds to wait after the last
// message arrived before a batch is flushed automatically.
// By default this parameter is set to "5".
//
// - Batch/MaxCount: This value defines the maximum number of messages that can be buffered
// before a flush is mandatory. If the buffer is full and a flush is still
// underway or cannot be triggered out of other reasons, the producer will block.
// By default this parameter is set to "8192".
//
// - Batch/FlushCount: This value defines the number of messages to be buffered before they are
// written to disk. This setting is clamped to "BatchMaxCount".
// By default this parameter is set to "`BatchMaxCount` / 2".
//
// - Batch/FlushTimeoutSec: This value defines the maximum number of seconds to wait before
// a flush is aborted during shutdown. Set this parameter to "0" which does not abort
// the flushing procedure.
// By default this parameter is set to "0".
//
type BatchedWriterConfig struct {
	BatchTimeout      time.Duration `config:"Batch/TimeoutSec" default:"5" metric:"sec"`
	BatchMaxCount     int           `config:"Batch/MaxCount" default:"8192"`
	BatchFlushCount   int           `config:"Batch/FlushCount" default:"4096"`
	BatchFlushTimeout time.Duration `config:"Batch/FlushTimeoutSec" default:"0" metric:"sec"`
}

// Configure interface implementation
func (c *BatchedWriterConfig) Configure(conf core.PluginConfigReader) {
	c.BatchFlushCount = tmath.MinI(c.BatchFlushCount, c.BatchMaxCount)
}

// BatchedWriterAssembly is a helper struct for io.Writer compatible classes that use batch directly for resources
type BatchedWriterAssembly struct {
	Batch    core.MessageBatch // Batch contains the MessageBatch
	Created  time.Time         // Created contains the creation time from the writer was set
	config   BatchedWriterConfig
	writer   BatchedWriter
	assembly core.WriterAssembly
	logger   logrus.FieldLogger
}

// BatchedWriter is an interface for different file writer like disk, s3, etc.
type BatchedWriter interface {
	io.WriteCloser
	Name() string // base name of the file/resource
	Size() int64  // length in bytes for regular files; system-dependent for others
	IsAccessible() bool
}

// NewBatchedWriterAssembly returns a new BatchedWriterAssembly instance
func NewBatchedWriterAssembly(config BatchedWriterConfig, modulator core.Modulator, tryFallback func(*core.Message), logger logrus.FieldLogger) *BatchedWriterAssembly {
	return &BatchedWriterAssembly{
		Batch:    core.NewMessageBatch(config.BatchMaxCount),
		assembly: core.NewWriterAssembly(nil, tryFallback, modulator),
		config:   config,
		logger:   logger,
	}
}

// HasWriter returns boolean value if a writer i currently set
func (bwa *BatchedWriterAssembly) HasWriter() bool {
	return bwa.writer != nil
}

// SetWriter set a BatchedWriter interface implementation
func (bwa *BatchedWriterAssembly) SetWriter(writer BatchedWriter) {
	bwa.writer = writer
	bwa.Created = time.Now()
}

// UnsetWriter unset the current writer
func (bwa *BatchedWriterAssembly) UnsetWriter() {
	bwa.writer = nil
}

// GetWriterAndUnset returns the current writer and unset it
func (bwa *BatchedWriterAssembly) GetWriterAndUnset() BatchedWriter {
	writer := bwa.GetWriter()
	bwa.UnsetWriter()
	return writer
}

// GetWriter returns the current writer
func (bwa *BatchedWriterAssembly) GetWriter() BatchedWriter {
	return bwa.writer
}

// Flush flush the batch
func (bwa *BatchedWriterAssembly) Flush() {
	if bwa.writer != nil {
		bwa.assembly.SetWriter(bwa.writer)
		bwa.Batch.Flush(bwa.assembly.Write)
	} else {
		bwa.Batch.Flush(bwa.assembly.Flush)
	}
}

// Close closes batch and writer
func (bwa *BatchedWriterAssembly) Close() {
	if bwa.writer != nil {
		bwa.assembly.SetWriter(bwa.writer)
		bwa.Batch.Close(bwa.assembly.Write, bwa.config.BatchFlushTimeout)
	} else {
		bwa.Batch.Close(bwa.assembly.Flush, bwa.config.BatchFlushTimeout)
	}
	bwa.writer.Close()
}

// FlushOnTimeOut checks if timeout or slush count reached and flush in this case
func (bwa *BatchedWriterAssembly) FlushOnTimeOut() {
	if bwa.Batch.ReachedTimeThreshold(bwa.config.BatchTimeout) || bwa.Batch.ReachedSizeThreshold(bwa.config.BatchFlushCount) {
		bwa.Flush()
	}
}

// NeedsRotate evaluate if the BatchedWriterAssembly need to rotate by the FileRotateConfig
func (bwa *BatchedWriterAssembly) NeedsRotate(rotate RotateConfig, forceRotate bool) (bool, error) {
	// File does not exist?
	if !bwa.HasWriter() {
		bwa.logger.Debug("Rotate true: ", "no writer")
		return true, nil
	}

	// File can be accessed?
	if !bwa.GetWriter().IsAccessible() {
		bwa.logger.Debug("Rotate false: ", "no access")
		return false, errors.New("Can' access file to rotate")
	}

	// File needs rotation?
	if !rotate.Enabled {
		bwa.logger.Debug("Rotate false: ", "not active")
		return false, nil
	}

	if forceRotate {
		bwa.logger.Debug("Rotate true: ", "forced")
		return true, nil
	}

	// File is too large?
	if bwa.GetWriter().Size() >= rotate.SizeByte {
		bwa.logger.Debug("Rotate true: ", "size > rotation size")
		return true, nil // ### return, too large ###
	}

	// File is too old?
	if time.Since(bwa.Created) >= rotate.Timeout {
		bwa.logger.Debug("Rotate true: ", "lifetime > timeout setting")
		return true, nil // ### return, too old ###
	}

	// RotateAt crossed?
	if rotate.AtHour > -1 && rotate.AtMinute > -1 {
		now := time.Now()
		rotateAt := time.Date(now.Year(), now.Month(), now.Day(), rotate.AtHour, rotate.AtMinute, 0, 0, now.Location())

		if rotateAt.Before(bwa.Created) {
			return false, nil
		}

		if now.After(rotateAt) {
			bwa.logger.Debug("Rotate true: ", "rotateAt time reached")
			return true, nil // ### return, too old ###
		}
	}

	// nope, everything is ok
	return false, nil
}
