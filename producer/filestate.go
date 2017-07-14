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
	"github.com/sirupsen/logrus"
	"github.com/trivago/gollum/core"
	"io"
	"time"
)

// BatchedWriterAssembly is a helper struct for io.Writer compatible classes that use batch directly for resources
type BatchedWriterAssembly struct {
	writer          BatchedWriter
	batch           core.MessageBatch
	assembly        core.WriterAssembly
	created         time.Time
	flushTimeout    time.Duration // max sec to wait before a flush is aborted
	batchTimeout    time.Duration // max sec to wait before batch will flushed
	batchFlushCount int
	logger          logrus.FieldLogger
}

// BatchedWriter is an interface for different file writer like disk, s3, etc.
// TODO: Replace methods with os.FileInfo interface
type BatchedWriter interface {
	io.WriteCloser
	Name() string // base name of the file/resource
	Size() int64  // length in bytes for regular files; system-dependent for others
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
