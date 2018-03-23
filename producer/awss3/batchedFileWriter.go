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
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/sirupsen/logrus"
	"github.com/trivago/gollum/core/components"
)

const minUploadPartSize = 5 * 1024 * 1024

// BatchedFileWriterInterface extends the components.BatchedWriter interface for rotation checks
type BatchedFileWriterInterface interface {
	components.BatchedWriter
	GetUploadCount() int
}

// BatchedFileWriter is the file producer core.BatchedWriter implementation for the core.BatchedWriterAssembly
type BatchedFileWriter struct {
	s3Client    *s3.S3
	s3Bucket    string
	s3SubFolder string
	fileName    string
	logger      logrus.FieldLogger

	currentMultiPart int64               // current multipart count
	s3UploadID       *string             // upload id from s3 for active file
	totalSize        int                 // total size off all writes to this writer (need for rotations)
	completedParts   []*s3.CompletedPart // collection of uploaded parts

	// need separate byte buffer for min 5mb part uploads.
	// @see http://docs.aws.amazon.com/AmazonS3/latest/API/mpUploadComplete.html
	activeBuffer *s3ByteBuffer
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

	batchedFileWriter := BatchedFileWriter{
		s3Client:    s3Client,
		s3Bucket:    s3Bucket,
		s3SubFolder: s3SubFolder,
		fileName:    fileName,
		logger:      logger,
	}

	batchedFileWriter.init()
	return batchedFileWriter
}

// init BatchedFileWriter struct
func (w *BatchedFileWriter) init() {
	w.totalSize = 0
	w.currentMultiPart = 0
	w.completedParts = []*s3.CompletedPart{}
	w.activeBuffer = newS3ByteBuffer()

	w.createMultipartUpload()
}

// Write is part of the BatchedWriter interface and wraps the file.Write() implementation
func (w *BatchedFileWriter) Write(p []byte) (n int, err error) {
	w.activeBuffer.Write(p)

	if size, _ := w.activeBuffer.Size(); size >= minUploadPartSize {
		w.logger.WithField("size", size).Debug("Buffer size ready for request")
		w.uploadPartInput()
	} else {
		w.logger.WithField("size", size).Debug("Buffer size not big enough vor request")
	}

	length := len(p)
	w.totalSize += length
	return length, nil
}

// Name is part of the BatchedWriter interface and wraps the file.Name() implementation
func (w *BatchedFileWriter) Name() string {
	return w.fileName
}

// Size is part of the BatchedWriter interface and wraps the file.Stat().Size() implementation
func (w *BatchedFileWriter) Size() int64 {
	return int64(w.totalSize)
}

// IsAccessible is part of the BatchedWriter interface and check if the writer can access his file
func (w *BatchedFileWriter) IsAccessible() bool {
	return w.s3UploadID != nil
}

// Close is part of the Close interface and handle the file close or compression call
func (w *BatchedFileWriter) Close() error {
	// flush upload buffer
	err := w.uploadPartInput()

	w.completeMultipartUpload()

	return err
}

// GetUploadCount returns the count of completed part uploads
func (w *BatchedFileWriter) GetUploadCount() int {
	return len(w.completedParts)
}

func (w *BatchedFileWriter) getS3Path() string {
	if w.s3SubFolder != "" {
		return fmt.Sprintf("%s/%s", w.s3SubFolder, w.Name())
	}

	return w.Name()
}

func (w *BatchedFileWriter) uploadPartInput() (err error) {
	if size, _ := w.activeBuffer.Size(); size < 1 {
		w.logger.Warning("uploadPartInput(): empty buffer - no upload necessary")
		return
	}

	// increase and get currentMultiPart count
	w.currentMultiPart++
	currentMultiPart := w.currentMultiPart

	// get and reset active buffer
	buffer := w.activeBuffer
	w.activeBuffer = newS3ByteBuffer()

	input := &s3.UploadPartInput{
		Body:       buffer,
		Bucket:     aws.String(w.s3Bucket),
		Key:        aws.String(w.getS3Path()),
		PartNumber: aws.Int64(currentMultiPart),
		UploadId:   w.s3UploadID,
	}

	result, err := w.s3Client.UploadPart(input)
	if err != nil {
		w.logger.WithError(err).WithField("input", input).Errorf("Can't upload part '%d'", currentMultiPart)
		return
	}

	w.logger.
		WithField("part", currentMultiPart).
		WithField("result", result).
		Debug("upload part successfully send")

	completedPart := s3.CompletedPart{}
	completedPart.SetETag(*result.ETag)
	completedPart.SetPartNumber(currentMultiPart)

	w.completedParts = append(w.completedParts, &completedPart)

	return
}

func (w *BatchedFileWriter) createMultipartUpload() {
	input := &s3.CreateMultipartUploadInput{
		Bucket: aws.String(w.s3Bucket),
		Key:    aws.String(w.getS3Path()),
	}

	result, err := w.s3Client.CreateMultipartUpload(input)
	if err != nil {
		w.logger.WithError(err).WithField("file", w.Name()).Error("Can't create multipart upload")
		return
	}

	w.s3UploadID = result.UploadId
	w.logger.WithField("uploadId", result.UploadId).Debug("successfully created multipart upload")
}

func (w *BatchedFileWriter) completeMultipartUpload() {
	if w.currentMultiPart < 1 {
		w.logger.Warning("No completeMultipartUpload request necessary for zero parts")
		return
	}

	input := &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(w.s3Bucket),
		Key:      aws.String(w.getS3Path()),
		UploadId: w.s3UploadID,
		MultipartUpload: &s3.CompletedMultipartUpload{
			Parts: w.completedParts,
		},
	}

	result, err := w.s3Client.CompleteMultipartUpload(input)
	if err != nil {
		w.logger.WithError(err).
			WithField("input", input).
			WithField("response", result).
			Error("Can't complete multipart upload")
		return
	}

	w.logger.
		WithField("location", *result.Location).
		WithField("parts", len(w.completedParts)).
		Debug("successfully completed MultipartUpload")
	w.s3UploadID = nil // reset upload id
}
