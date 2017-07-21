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
	"fmt"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/components"
	"github.com/trivago/gollum/producer/awsS3"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const defaultAwsEndpoint = "s3.amazonaws.com"

// AwsS3 producer plugin
//
// Configuration example
//
// myProducer:
//   Type: producer.AwsS3
//   Credentials:
//     Type: shared
//     File: /Users/$MYUSER/.aws/credentials
//     Profile: default
//   Region: eu-west-1
//   Bucket: my-s3-bucket/subfolder/
//   Streams:
//     - "*"
//   File: gollum_*.log
//   Batch:
//     TimeoutSec: 5
//     MaxCount: 1000
//     FlushCount: 500
//     FlushTimeoutSec: 0
//   Rotation:
//     TimestampFormat: 2006-01-02_15
//     TimeoutMin: 1440
//     SizeMB: 1024
// 	   ZeroPadding: 0
// 	   At: 13:05
//   Modulators:
//     - format.Envelope:
//       Postfix: "\n"
//
type AwsS3 struct {
	core.DirectProducer `gollumdoc:"embed_type"`

	// Rotate is public to make Pruner.Configure() callable (bug in treflect package)
	// AwsMultiClient is public to make AwsMultiClient.Configure() callable (bug in treflect package)
	// BatchConfig is public to make BatchedWriterConfig.Configure() callable (bug in treflect package)
	Rotate         components.RotateConfig        `gollumdoc:"embed_type"`
	AwsMultiClient components.AwsMultiClient      `gollumdoc:"embed_type"`
	BatchConfig    components.BatchedWriterConfig `gollumdoc:"embed_type"`

	// configurations
	bucket          string `config:"Bucket" default:""`
	fileNamePattern string `config:"File" default:"gollum_*.log"`

	// properties
	filesByStream    map[core.MessageStreamID]*components.BatchedWriterAssembly
	files            map[string]*components.BatchedWriterAssembly
	hasWildcard      bool
	batchedFileGuard *sync.RWMutex
	s3Client         *s3.S3
}

func init() {
	core.TypeRegistry.Register(AwsS3{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *AwsS3) Configure(conf core.PluginConfigReader) {
	prod.SetRollCallback(prod.rotateTargetFiles)
	prod.SetStopCallback(prod.close)

	prod.filesByStream = make(map[core.MessageStreamID]*components.BatchedWriterAssembly)
	prod.files = make(map[string]*components.BatchedWriterAssembly)

	prod.hasWildcard = strings.IndexByte(prod.fileNamePattern, '*') != -1
	prod.Rotate.Enabled = true // force rotation

	prod.batchedFileGuard = new(sync.RWMutex)
}

// Produce writes to a buffer that is dumped to a file.
func (prod *AwsS3) Produce(workers *sync.WaitGroup) {
	prod.initS3Client()

	prod.AddMainWorker(workers)
	prod.TickerMessageControlLoop(prod.writeMessage, prod.BatchConfig.BatchTimeout, prod.writeBatchOnTimeOut)
}

func (prod *AwsS3) initS3Client() {
	sess, err := prod.AwsMultiClient.NewSessionWithOptions()
	if err != nil {
		prod.Logger.WithError(err).Error("Can't get proper aws config")
	}

	awsConfig := prod.AwsMultiClient.GetConfig()

	// set auto endpoint to s3 if setting is empty
	if awsConfig.Endpoint == nil || *awsConfig.Endpoint == "" {
		if *awsConfig.Region != components.DefaultAwsRegion {
			awsConfig.WithEndpoint(fmt.Sprintf("s3-%s.amazonaws.com", *awsConfig.Region))
		} else {
			awsConfig.WithEndpoint(defaultAwsEndpoint)
		}
	}

	prod.s3Client = s3.New(sess, awsConfig)
}

func (prod *AwsS3) getBatchedFile(streamID core.MessageStreamID, forceRotate bool) (*components.BatchedWriterAssembly, error) {
	// get batchedFile from filesByStream[streamID] map
	prod.batchedFileGuard.RLock()
	batchedFile, fileExists := prod.filesByStream[streamID]
	prod.batchedFileGuard.RUnlock()
	if fileExists {
		if rotate, err := prod.needsRotate(batchedFile, forceRotate); !rotate {
			return batchedFile, err // ### return, already open or error ###
		}
	}

	prod.batchedFileGuard.Lock()
	defer prod.batchedFileGuard.Unlock()

	// check again to avoid race conditions
	if batchedFile, fileExists := prod.filesByStream[streamID]; fileExists {
		if rotate, err := prod.needsRotate(batchedFile, forceRotate); !rotate {
			return batchedFile, err // ### return, already open or error ###
		}
	}

	baseFileName := prod.getBaseFileName(streamID)

	// get batchedFile from files[path] and assure the file is correctly mapped
	batchedFile, fileExists = prod.files[baseFileName]
	if !fileExists {
		batchedFile = prod.newBatchedWriterAssembly()

		prod.filesByStream[streamID] = batchedFile
		prod.files[baseFileName] = batchedFile
	}

	// Close existing batchedFile.writer
	if batchedFile.HasWriter() {
		oldAwsWriter := batchedFile.GetWriterAndUnset()

		prod.Logger.Info("Rotated ", oldAwsWriter.Name(), " -> ", baseFileName)
		go oldAwsWriter.Close() // close in subroutine for eventually compression in the background
	}

	// Update BatchedWriterAssembly writer
	writer := awsS3.NewBatchedFileWriter(prod.s3Client, prod.bucket, prod.getFinalFileName(baseFileName), prod.Logger)
	batchedFile.SetWriter(&writer)

	return batchedFile, nil
}

func (prod *AwsS3) needsRotate(batchedFile *components.BatchedWriterAssembly, forceRotate bool) (bool, error) {
	// run default rotation checks
	if needUpload, err := batchedFile.NeedsRotate(prod.Rotate, forceRotate); needUpload {
		return true, err
	}

	// check if max multipart uploads of 1000 reached
	// @see: http://docs.aws.amazon.com/AmazonS3/latest/dev/mpuoverview.html
	// we use 995 to have a small buffer to the limit and need at least +1 upload part for the last flush
	writer, ok := batchedFile.GetWriter().(awsS3.BatchedFileWriterInterface)
	if ok && writer.GetUploadCount() > 995 {
		prod.Logger.Debug("Rotate true: ", "upload count reached limit of 1000")
		return true, nil
	}

	return false, nil
}

func (prod *AwsS3) getBaseFileName(streamID core.MessageStreamID) string {
	if prod.hasWildcard {
		streamName := core.StreamRegistry.GetStreamName(streamID)
		return strings.Replace(prod.fileNamePattern, "*", streamName, -1)
	}

	return prod.fileNamePattern
}

func (prod *AwsS3) getFinalFileName(baseFileName string) string {
	fileExt := filepath.Ext(baseFileName)
	fileName := baseFileName[:len(baseFileName)-len(fileExt)]

	timestamp := time.Now().Format(prod.Rotate.Timestamp)
	signature := fmt.Sprintf("%s_%s", fileName, timestamp)
	maxSuffix := uint64(0)

	//todo: get list from aws?
	/*files, _ := ioutil.ReadDir(streamFile.dir)
	for _, f := range files {
		if strings.HasPrefix(f.Name(), signature) {
			// Special case.
			// If there is no extension, counter stays at 0
			// If there is an extension (and no count), parsing the "." will yield a counter of 0
			// If there is a count, parsing it will work as intended
			counter := uint64(0)
			if len(f.Name()) > len(signature) {
				counter, _ = tstrings.Btoi([]byte(f.Name()[len(signature)+1:]))
			}

			if maxSuffix <= counter {
				maxSuffix = counter + 1
			}
		}
	}*/

	if maxSuffix == 0 {
		return fmt.Sprintf("%s%s", signature, fileExt)
	}

	formatString := "%s_%d%s"
	if prod.Rotate.ZeroPad > 0 {
		formatString = fmt.Sprintf("%%s_%%0%dd%%s", prod.Rotate.ZeroPad)
	}
	return fmt.Sprintf(formatString, signature, int(maxSuffix), fileExt)

}

func (prod *AwsS3) newBatchedWriterAssembly() *components.BatchedWriterAssembly {
	batchedFile := components.NewBatchedWriterAssembly(
		prod.BatchConfig,
		prod,
		prod.TryFallback,
		prod.Logger,
	)

	return batchedFile
}

func (prod *AwsS3) writeMessage(msg *core.Message) {
	batchedFile, err := prod.getBatchedFile(msg.GetStreamID(), false)
	if err != nil {
		prod.Logger.Error("Write error: ", err)
		prod.TryFallback(msg)
		return // ### return, fallback ###
	}

	batchedFile.Batch.AppendOrFlush(msg, batchedFile.Flush, prod.IsActiveOrStopping, prod.TryFallback)
}

func (prod *AwsS3) writeBatchOnTimeOut() {
	for _, batchedFile := range prod.files {
		batchedFile.FlushOnTimeOut()
	}
}

func (prod *AwsS3) rotateTargetFiles() {
	for streamID := range prod.filesByStream {
		if _, err := prod.getBatchedFile(streamID, true); err != nil {
			prod.Logger.Error("Rotate error: ", err)
		}
	}
}

func (prod *AwsS3) close() {
	defer prod.WorkerDone()

	for _, batchedFile := range prod.files {
		batchedFile.Close()
	}
}
