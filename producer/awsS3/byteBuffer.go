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

package awss3

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
)

// s3ByteBuffer is a byte buffer used for s3 target objects
type s3ByteBuffer struct {
	bytes    []byte
	position int64
}

func newS3ByteBuffer() *s3ByteBuffer {
	return &s3ByteBuffer{
		bytes:    make([]byte, 0),
		position: int64(0),
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
