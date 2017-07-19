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

package awsS3

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/sirupsen/logrus"
	"strings"
)

// BatchedFileWriter is the file producer core.BatchedWriter implementation for the core.BatchedWriterAssembly
type BatchedFileWriter struct {
	s3Client    *s3.S3
	s3Bucket    string
	s3SubFolder string
	fileName    string
	logger      logrus.FieldLogger
}

// NewBatchedFileWriter returns a BatchedFileWriter instance
func NewBatchedFileWriter(s3Client *s3.S3, bucket string, fileName string, logger logrus.FieldLogger) BatchedFileWriter {
	var s3Bucket, s3SubFolder string

	if strings.Contains(bucket, "/") {
		split := strings.SplitN(bucket, "/", 2)
		s3Bucket, s3SubFolder = split[0], split[1]
	} else {
		s3Bucket = bucket
		s3SubFolder = ""
	}

	return BatchedFileWriter{
		s3Client,
		s3Bucket,
		s3SubFolder,
		fileName,
		logger,
	}

	// CreateMultipartUpload (init)
	// UploadPart .. UploadPart .. UploadPart (write)
	// CompleteMultipartUpload (close)
}

// Write is part of the BatchedWriter interface and wraps the file.Write() implementation
func (w *BatchedFileWriter) Write(p []byte) (n int, err error) {
	w.logger.Warning("START MULTI UPLOAD AND WRITE TO AWS")

	byteBuffer := newS3ByteBuffer()
	byteBuffer.Write(p)

	param := &s3.PutObjectInput{
		Bucket:       aws.String(w.s3Bucket),
		Key:          aws.String(w.getS3Path()),
		Body:         byteBuffer,
		StorageClass: aws.String("STANDARD"), //TODO: make StorageClass configurable
	}

	rsp, err := w.s3Client.PutObject(param)
	if err != nil {
		w.logger.WithField("response", rsp.GoString()).WithError(err).Errorf("Failed to put object %s/%s", w.s3Bucket, w.Name())
		return 0, err
	}

	//todo: save size to property

	return len(p), nil
}

// Name is part of the BatchedWriter interface and wraps the file.Name() implementation
func (w *BatchedFileWriter) Name() string {
	return w.fileName
}

// Size is part of the BatchedWriter interface and wraps the file.Stat().Size() implementation
func (w *BatchedFileWriter) Size() int64 {
	w.logger.Warning("GET SIZE OF ALL BYTE WHICH UPLOADED")
	return 1000
}

// IsAccessible is part of the BatchedWriter interface and check if the writer can access his file
func (w *BatchedFileWriter) IsAccessible() bool {
	w.logger.Warning("CHECK FILE ACCESS")

	return true
}

// Close is part of the Close interface and handle the file close or compression call
func (w *BatchedFileWriter) Close() error {
	w.logger.Warning("CLOSE MULTI UPLOAD")
	w.logger.Warning("CLOSE CONNECTION")

	return nil
}

func (w *BatchedFileWriter) getS3Path() string {
	if w.s3SubFolder != "" {
		return fmt.Sprintf("%s/%s", w.s3SubFolder, w.Name())
	}

	return w.Name()
}
