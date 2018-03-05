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

package deprecated

import (
	"compress/gzip"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/trivago/tgo/tsync"
	"io"
	"os"
)

type s3Buffer interface {
	Bytes() ([]byte, error)
	CloseAndDelete() error
	Compress() error
	Read(p []byte) (n int, err error)
	Seek(offset int64, whence int) (int64, error)
	Size() (int, error)
	Sha1() (string, error)
	Write(p []byte) (n int, err error)
}

// s3 buffer backed with a byte slice
type s3ByteBuffer struct {
	bytes    []byte
	position int64
	logger   logrus.FieldLogger
}

func newS3ByteBuffer(logger logrus.FieldLogger) *s3ByteBuffer {
	return &s3ByteBuffer{
		bytes:    make([]byte, 0),
		position: int64(0),
		logger:   logger,
	}
}

func (buf *s3ByteBuffer) Bytes() ([]byte, error) {
	return buf.bytes, nil
}

func (buf *s3ByteBuffer) CloseAndDelete() error {
	buf.bytes = make([]byte, 0)
	buf.position = 0
	return nil
}

func (buf *s3ByteBuffer) Read(p []byte) (n int, err error) {
	n = copy(p, buf.bytes[buf.position:])
	buf.position += int64(n)
	if buf.position == int64(len(buf.bytes)) {
		return n, io.EOF
	}
	return n, nil
}

func (buf *s3ByteBuffer) Write(p []byte) (n int, err error) {
	buf.bytes = append(buf.bytes[:buf.position], p...)
	buf.position += int64(len(p))
	return len(p), nil
}

func (buf *s3ByteBuffer) Seek(offset int64, whence int) (int64, error) {
	var position int64
	switch whence {
	case 0: // io.SeekStart
		position = offset
	case 1: // io.SeekCurrent
		position = buf.position + offset
	case 2: // io.SeekEnd
		position = int64(len(buf.bytes)) + offset
	}
	if position < 0 {
		return 0, fmt.Errorf("S3Buffer bad seek result %d", position)
	}
	buf.position = position
	return position, nil
}

func (buf *s3ByteBuffer) Size() (int, error) {
	return len(buf.bytes), nil
}

func (buf *s3ByteBuffer) Sha1() (string, error) {
	hash := sha1.Sum(buf.bytes)
	return hex.EncodeToString(hash[:]), nil
}

func (buf *s3ByteBuffer) Compress() error {
	compressed := newS3ByteBuffer(buf.logger)
	gzipWriter := gzip.NewWriter(compressed)
	_, err := gzipWriter.Write(buf.bytes)
	gzipWriter.Close()
	if err != nil {
		buf.logger.Warning("Compression failed:", err)
		return err
	}
	buf.bytes = compressed.bytes
	buf.position = compressed.position
	return nil
}

// s3 buffer backed with a file
type s3FileBuffer struct {
	file   *os.File
	logger logrus.FieldLogger
}

func newS3FileBuffer(filename string, logger logrus.FieldLogger) (*s3FileBuffer, error) {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		logger.Error("s3FileBuffer could not open file:", err)
		return nil, err
	}
	return &s3FileBuffer{
		file:   file,
		logger: logger,
	}, nil
}

func (buf *s3FileBuffer) Bytes() ([]byte, error) {
	size, err := buf.Size()
	if err != nil {
		return nil, err
	}
	bytes := make([]byte, size)
	_, err = buf.Read(bytes)
	if err != nil && err != io.EOF {
		buf.logger.Error("s3FileBuffer.Bytes() read error: ", err)
		return nil, err
	}
	return bytes, nil
}

func (buf *s3FileBuffer) CloseAndDelete() error {
	filename := buf.file.Name()
	if err := buf.file.Close(); err != nil {
		buf.logger.Error("s3FileBuffer.CloseAndDelete() failed to close file:", err)
		return err
	}
	if err := os.Remove(filename); err != nil {
		buf.logger.Error("s3FileBuffer.CloseAndDelete() failed to rm file:", err)
		return err
	}
	return nil
}

func (buf *s3FileBuffer) Read(p []byte) (n int, err error) {
	return buf.file.Read(p)
}

func (buf *s3FileBuffer) Seek(offset int64, whence int) (int64, error) {
	return buf.file.Seek(offset, whence)
}

func (buf *s3FileBuffer) Write(p []byte) (n int, err error) {
	return buf.file.Write(p)
}

func (buf *s3FileBuffer) Sha1() (string, error) {
	hasher := sha1.New()
	buf.file.Seek(0, 0)
	_, err := io.Copy(hasher, buf.file)
	if err != nil {
		buf.logger.Error("s3FileBuffer.Sha1() hashing error: ", err)
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)[:]), nil
}

func (buf *s3FileBuffer) Size() (int, error) {
	stats, err := buf.file.Stat()
	if err != nil {
		buf.logger.Error("s3FileBuffer.Size() could not stat file:", err)
		return 0, err
	}
	return int(stats.Size()), err
}

func (buf *s3FileBuffer) Compress() error {
	// Generate file to gzip into
	filename := buf.file.Name() + ".gz"
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		buf.logger.Error("s3FileBuffer.Compress() could not open file:", err)
		return err
	}

	gzipWriter := gzip.NewWriter(file)
	spin := tsync.NewSpinner(tsync.SpinPriorityHigh)
	buf.file.Seek(0, 0)

	for err == nil {
		_, err = io.CopyN(gzipWriter, buf.file, 1<<20) // 1 MB chunks
		spin.Yield()                                   // Be async!
	}
	gzipWriter.Close()

	if err != nil && err != io.EOF {
		buf.logger.Warning("s3FileBuffer.Compress() failed to compress file:", err)
		file.Close()
		if err2 := os.Remove(filename); err2 != nil {
			buf.logger.Error("s3FileBuffer.Compress() failed to rm compressed file:", err)
		}
		return err
	}

	// cleanup after compression
	buf.file.Close()
	err = os.Remove(buf.file.Name())
	if err != nil {
		// don't return this error, because compression succeeded
		buf.logger.Warning("s3FileBuffer.Compress() failed to rm original file:", err)
	}

	buf.file = file
	return nil
}
