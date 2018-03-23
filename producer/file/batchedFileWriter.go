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

package file

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/trivago/tgo/tio"
	"github.com/trivago/tgo/tsync"
)

// BatchedFileWriter is the file producer core.BatchedWriter implementation for the core.BatchedWriterAssembly
type BatchedFileWriter struct {
	file            *os.File
	compressOnClose bool
	stats           os.FileInfo
	logger          logrus.FieldLogger
}

// NewBatchedFileWriter returns a BatchedFileWriter instance
func NewBatchedFileWriter(file *os.File, compressOnClose bool, logger logrus.FieldLogger) BatchedFileWriter {
	return BatchedFileWriter{
		file,
		compressOnClose,
		nil,
		logger,
	}
}

// Write is part of the BatchedWriter interface and wraps the file.Write() implementation
func (w *BatchedFileWriter) Write(p []byte) (n int, err error) {
	return w.file.Write(p)
}

// Name is part of the BatchedWriter interface and wraps the file.Name() implementation
func (w *BatchedFileWriter) Name() string {
	return w.file.Name()
}

// Size is part of the BatchedWriter interface and wraps the file.Stat().Size() implementation
func (w *BatchedFileWriter) Size() int64 {
	stats, err := w.getStats()
	if err != nil {
		return 0
	}
	return stats.Size()
}

// IsAccessible is part of the BatchedWriter interface and check if the writer can access his file
func (w *BatchedFileWriter) IsAccessible() bool {
	_, err := w.getStats()
	return err == nil
}

// Close is part of the Close interface and handle the file close or compression call
func (w *BatchedFileWriter) Close() error {
	if w.compressOnClose {
		return w.compressAndCloseLog()
	}

	return w.file.Close()
}

func (w *BatchedFileWriter) getStats() (os.FileInfo, error) {
	if w.stats != nil {
		return w.stats, nil
	}

	stats, err := w.file.Stat()
	if err != nil {
		return nil, err
	}

	w.stats = stats
	return w.stats, nil
}

func (w *BatchedFileWriter) compressAndCloseLog() error {
	// Generate file to zip into
	sourceFileName := w.Name()
	sourceDir, sourceBase, _ := tio.SplitPath(sourceFileName)

	targetFileName := fmt.Sprintf("%s/%s.gz", sourceDir, sourceBase)

	targetFile, err := os.OpenFile(targetFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		w.logger.Error("Compress error:", err)
		w.file.Close()
		return err
	}

	// Create zipfile and compress data
	w.logger.Info("Compressing " + sourceFileName)

	w.file.Seek(0, 0)
	targetWriter := gzip.NewWriter(targetFile)
	spin := tsync.NewSpinner(tsync.SpinPriorityHigh)

	for err == nil {
		_, err = io.CopyN(targetWriter, w.file, 1<<20) // 1 MB chunks
		spin.Yield()                                   // Be async!
	}

	// Cleanup
	w.file.Close()
	targetWriter.Close()
	targetFile.Close()

	if err != nil && err != io.EOF {
		w.logger.Warning("Compression failed:", err)
		err = os.Remove(targetFileName)
		if err != nil {
			w.logger.Error("Compressed file remove failed:", err)
		}
		return err
	}

	// Remove original log
	err = os.Remove(sourceFileName)
	if err != nil {
		w.logger.Error("Uncompressed file remove failed:", err)
		return err
	}

	return nil
}
