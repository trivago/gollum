// Copyright 2015-2016 trivago GmbH
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
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	s3CredentialEnv    = "environment"
	s3CredentialStatic = "static"
	s3CredentialShared = "shared"
	s3CredentialNone   = "none"
)

// S3 producer plugin
// This producer sends data to an AWS S3 Bucket.
// Configuration example
//
//  - "producer.S3":
//    Region: "eu-west-1"
//    Endpoint: "s3-eu-west-1.amazonaws.com"
//    StorageClass: "STANDARD"
//    CredentialType: "none"
//    CredentialId: ""
//    CredentialToken: ""
//    CredentialSecret: ""
//    CredentialFile: ""
//    CredentialProfile: ""
//    BatchMaxMessages: 5000
//    ObjectMaxMessages: 5000
//    ObjectMessageDelimiter: "\n"
//    SendTimeframeMs: 10000
//    BatchTimeoutSec: 30
//    TimestampWrite: "2006-01-02T15:04:05"
//    PathFormatter: ""
//    Compress: false
//    LocalPath: ""
//    UploadOnShutdown: false
//    FileMaxAgeSec: 3600
//    FileMaxMB: 1000
//    StreamMapping:
//      "*" : "bucket/path"
//
// Region defines the amazon region of your s3 bucket.
// By default this is set to "eu-west-1".
//
// Endpoint defines the amazon endpoint for your s3 bucket.
// By default this is set to "s3-eu-west-1.amazonaws.com"
//
// StorageClass defines the amazon s3 storage class for objects created, from
// http://docs.aws.amazon.com/AmazonS3/latest/dev/storage-class-intro.html
// By default this is set to "STANDARD".
//
// CredentialType defines the credentials that are to be used when
// connectiong to kensis. This can be one of the following: environment,
// static, shared, none.
// Static enables the parameters CredentialId, CredentialToken and
// CredentialSecret shared enables the parameters CredentialFile and
// CredentialProfile. None will not use any credentials and environment
// will pull the credentials from environmental settings.
// By default this is set to none.
//
// BatchMaxMessages defines the maximum number of messages to upload per
// batch. By default this is set to 5000.
//
// ObjectMaxMessages defines the maximum number of messages to join into
// an s3 object. By default this is set to 5000.
//
// ObjectMessageDelimiter defines the string to delimit messages within
// an s3 object. By default this is set to "\n".
//
// SendTimeframeMs defines the timeframe in milliseconds in which a second
// batch send can be triggered. By default this is set to 10000, i.e. ten
// upload operations per second per s3 path.
//
// BatchTimeoutSec defines the number of seconds after which a batch is
// flushed automatically. By default this is set to 30.
//
// TimestampWrite defines the go timestamp format that will be used in naming
// objects. Objects are named <s3_path><timestamp><sha1>. By default timestamp
// is set to "2006-01-02T15:04:05".
//
// PathFormatter can define a formatter that extracts the path suffix for an s3
// object from the object data. By default this is uses the sha1 of the object.
// A good formatter for this can be format.Identifier.
//
// Compress defines whether to gzip compress the object before uploading.
// This adds a ".gz" extension to objects. By default this is set to false.
//
// LocalPath defines the local output directory for temporary object files.
// Files will be stored as "<path>/<number>". Compressed files will have a .gz
// extension. State will be stored in "<path>/state". By default this is
// not set, and objects will be built in memory.
//
// UploadOnShutdown defines whether to upload all temporary object files on
// shutdown. This has no effect if LocalPath is not set. By default this is false.
//
// FileMaxAgeSec defines the maximum age of a local file before it is uploaded.
// This defaults to 3600 (1 hour).
//
// FileMaxMB defines the maximum size of a local file before it is uploaded.
// This limit is imposed before compression occurs. This defaults to 1000 (1 GB).
//
// StreamMapping defines a translation from gollum stream to s3 bucket/path. If
// no mapping is given the gollum stream name is used as s3 bucket.
// Values are of the form bucket/path or bucket, s3:// prefix is not allowed.
// The full path of the object will be s3://<StreamMapping><Timestamp><PathFormat>
// where Timestamp is time the object is written formatted with TimestampWrite,
// and PathFormat is the output of PathFormatter when passed the object data.
type S3 struct {
	core.ProducerBase
	client            *s3.S3
	config            *aws.Config
	storageClass      string
	streamMap         map[core.MessageStreamID]string
	pathFormat        core.Formatter
	batch             core.MessageBatch
	objectMaxMessages int
	delimiter         []byte
	flushFrequency    time.Duration
	timeWrite         string
	lastSendTime      time.Time
	sendTimeLimit     time.Duration
	counters          map[string]*int64
	lastMetricUpdate  time.Time
	closing           bool
	compress          bool
	fileMaxAge        time.Duration
	fileMaxSize       int
	localPath         string
	nextFile          int64
	objects           map[string]*objectData
	objectsLock       *sync.Mutex
	stateFile         string
	uploadOnShutdown  bool
	useFiles          bool
}

const (
	s3MetricMessages    = "S3:Messages-"
	s3MetricMessagesSec = "S3:MessagesSec-"
)

type objectData struct {
	Compressed bool
	Created    time.Time
	Filename   string
	Messages   int
	S3Path     string
	Uploaded   bool
	buffer     s3Buffer
	lock       *sync.Mutex
}

func init() {
	shared.TypeRegistry.Register(S3{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *S3) Configure(conf core.PluginConfig) error {
	err := prod.ProducerBase.Configure(conf)
	if err != nil {
		return err
	}
	prod.SetStopCallback(prod.close)

	prod.storageClass = conf.GetString("StorageClass", "STANDARD")
	prod.streamMap = conf.GetStreamMap("StreamMapping", "")
	prod.batch = core.NewMessageBatch(conf.GetInt("BatchMaxMessages", 5000))
	prod.objectMaxMessages = conf.GetInt("ObjectMaxMessages", 5000)
	prod.delimiter = []byte(conf.GetString("ObjectMessageDelimiter", "\n"))
	prod.flushFrequency = time.Duration(conf.GetInt("BatchTimeoutSec", 30)) * time.Second
	prod.sendTimeLimit = time.Duration(conf.GetInt("SendTimeframeMs", 10000)) * time.Millisecond
	prod.timeWrite = conf.GetString("TimestampWrite", "2006-01-02T15:04:05")
	prod.lastSendTime = time.Now()
	prod.counters = make(map[string]*int64)
	prod.lastMetricUpdate = time.Now()
	prod.closing = false
	prod.compress = conf.GetBool("Compress", false)
	prod.objects = make(map[string]*objectData)
	prod.objectsLock = new(sync.Mutex)
	prod.uploadOnShutdown = conf.GetBool("UploadOnShutdown", false)
	prod.fileMaxSize = conf.GetInt("FileMaxMB", 1000) * 1000000
	prod.fileMaxAge = time.Duration(conf.GetInt("FileMaxAgeSec", 3600)) * time.Second

	for _, s3Path := range prod.streamMap {
		shared.Metric.New(s3MetricMessages + s3Path)
		shared.Metric.New(s3MetricMessagesSec + s3Path)
		prod.counters[s3Path] = new(int64)
	}

	prod.localPath = conf.GetString("LocalPath", "")
	prod.useFiles = prod.localPath != ""
	if prod.useFiles {
		if err := os.MkdirAll(prod.localPath, 0700); err != nil {
			return fmt.Errorf("Failed to create %s because of %s", prod.localPath, err.Error()) // ### return, missing directory ###
		}
		prod.stateFile = path.Join(prod.localPath, "state")
		if _, err := os.Stat(prod.stateFile); !os.IsNotExist(err) {
			data, err := ioutil.ReadFile(prod.stateFile)
			if err != nil {
				return err
			}
			if err := json.Unmarshal(data, &prod.objects); err != nil {
				return err
			}
			for s3Path, object := range prod.objects {
				filename := object.Filename
				basename := path.Base(filename)
				if basenum, err := strconv.ParseInt(basename, 10, 64); err == nil && basenum > prod.nextFile {
					prod.nextFile = basenum
				}
				if object.Compressed {
					filename += ".gz"
				}
				buffer, err := newS3FileBuffer(filename)
				if err != nil {
					return err
				}
				object.buffer = buffer
				object.lock = new(sync.Mutex)
				// add missing metrics
				if _, exists := prod.counters[s3Path]; !exists {
					shared.Metric.New(s3MetricMessages + s3Path)
					shared.Metric.New(s3MetricMessagesSec + s3Path)
					prod.counters[s3Path] = new(int64)
				}
			}
		}
	}

	if conf.HasValue("PathFormatter") {
		keyFormatter, err := core.NewPluginWithType(conf.GetString("PathFormatter", "format.Identifier"), conf)
		if err != nil {
			return err // ### return, plugin load error ###
		}
		prod.pathFormat = keyFormatter.(core.Formatter)
	} else {
		prod.pathFormat = nil
	}

	if prod.objectMaxMessages < 1 {
		prod.objectMaxMessages = 1
		Log.Warning.Print("ObjectMaxMessages was < 1. Defaulting to 1.")
	}

	if prod.objectMaxMessages > 1 && len(prod.delimiter) == 0 {
		prod.delimiter = []byte("\n")
		Log.Warning.Print("ObjectMessageDelimiter was empty. Defaulting to \"\\n\".")
	}

	// Config
	defaultEndpoint := "s3.amazonaws.com"
	prod.config = aws.NewConfig()
	if region := conf.GetString("Region", "eu-west-1"); region != "" {
		prod.config.WithRegion(region)
		if region != "us-east-1" {
			defaultEndpoint = "s3-" + region + ".amazonaws.com"
		}
	}

	if endpoint := conf.GetString("Endpoint", defaultEndpoint); endpoint != "" {
		prod.config.WithEndpoint(endpoint)
	}

	// Credentials
	credentialType := strings.ToLower(conf.GetString("CredentialType", s3CredentialNone))
	switch credentialType {
	case s3CredentialEnv:
		prod.config.WithCredentials(credentials.NewEnvCredentials())

	case s3CredentialStatic:
		id := conf.GetString("CredentialId", "")
		token := conf.GetString("CredentialToken", "")
		secret := conf.GetString("CredentialSecret", "")
		prod.config.WithCredentials(credentials.NewStaticCredentials(id, secret, token))

	case s3CredentialShared:
		filename := conf.GetString("CredentialFile", "")
		profile := conf.GetString("CredentialProfile", "")
		prod.config.WithCredentials(credentials.NewSharedCredentials(filename, profile))

	case s3CredentialNone:
		// Nothing

	default:
		return fmt.Errorf("Unknown CredentialType: %s", credentialType)
	}

	return nil
}

func (prod *S3) storeState() {
	if prod.useFiles {
		prod.objectsLock.Lock()
		defer prod.objectsLock.Unlock()
		data, err := json.Marshal(prod.objects)
		if err == nil {
			ioutil.WriteFile(prod.stateFile, data, 0600)
		}
	}
}

func (prod *S3) newS3Buffer() (buffer s3Buffer, filename string, err error) {
	if prod.useFiles {
		basename := atomic.AddInt64(&prod.nextFile, 1)
		filename := path.Join(prod.localPath, strconv.FormatInt(basename, 10))
		buffer, err := newS3FileBuffer(filename)
		return buffer, filename, err
	}

	return newS3ByteBuffer(), "", nil
}

func (prod *S3) bufferMessage(msg core.Message) {
	prod.batch.AppendOrFlush(msg, prod.sendBatch, prod.IsActiveOrStopping, prod.Drop)
}

func (prod *S3) sendBatchOnTimeOut() {
	// Flush if necessary
	if prod.batch.ReachedTimeThreshold(prod.flushFrequency) || prod.batch.ReachedSizeThreshold(prod.batch.Len()/2) {
		prod.sendBatch()
	}
	prod.uploadAllOnTimeout()
	prod.storeState()

	duration := time.Since(prod.lastMetricUpdate)
	prod.lastMetricUpdate = time.Now()

	for s3Path, counter := range prod.counters {
		count := atomic.SwapInt64(counter, 0)

		shared.Metric.Add(s3MetricMessages+s3Path, count)
		shared.Metric.SetF(s3MetricMessagesSec+s3Path, float64(count)/duration.Seconds())
	}
}

func (prod *S3) sendBatch() {
	prod.batch.Flush(prod.transformMessages)
}

func (prod *S3) upload(object *objectData, needLock bool) error {
	if needLock {
		object.lock.Lock()
		defer object.lock.Unlock()
	}

	if object.Uploaded || object.Messages == 0 {
		return nil
	}

	// compress
	if prod.compress && !object.Compressed {
		if err := object.buffer.Compress(); err != nil {
			return err
		}
		object.Compressed = true
	}

	// respect prod.sendTimeLimit
	sleepDuration := prod.sendTimeLimit - time.Since(prod.lastSendTime)
	if sleepDuration > 0 {
		time.Sleep(sleepDuration)
	}
	prod.lastSendTime = time.Now()

	// get bucket and key
	bucket, key := object.S3Path, ""
	if strings.Contains(object.S3Path, "/") {
		split := strings.SplitN(object.S3Path, "/", 2)
		bucket, key = split[0], split[1]
	}

	// add timestamp
	key += time.Now().Format(prod.timeWrite)

	// add pathFormat or sha1 suffix
	if prod.pathFormat != nil {
		data, err := object.buffer.Bytes()
		if err != nil {
			return err
		}
		msg := core.NewMessage(nil, data, uint64(0))
		byteKey, _ := prod.pathFormat.Format(msg)
		key += string(byteKey)
	} else {
		hash, err := object.buffer.Sha1()
		if err != nil {
			return err
		}
		key += hash
	}

	// .gz file extension
	if object.Compressed {
		key += ".gz"
	}

	// upload object.buffer
	params := &s3.PutObjectInput{
		Bucket:       aws.String(bucket),
		Key:          aws.String(key),
		Body:         object.buffer,
		StorageClass: aws.String(prod.storageClass),
	}
	_, err := prod.client.PutObject(params)
	atomic.AddInt64(prod.counters[object.S3Path], int64(1))
	if err != nil {
		Log.Error.Print("S3.upload() PutObject ", bucket+key, " error:", err)
		return err
	}

	// mark this object complete
	object.Uploaded = true
	return nil
}

func (prod *S3) needsUpload(object *objectData, nextMessageSize int) (bool, error) {
	upload := object.Messages >= prod.objectMaxMessages
	if prod.useFiles {
		size, err := object.buffer.Size()
		if err != nil {
			return false, err
		}
		upload = upload || (size+nextMessageSize >= prod.fileMaxSize)
		upload = upload || (time.Since(object.Created) >= prod.fileMaxAge)
	}
	upload = upload || (object.Compressed)
	return upload, nil
}

func (prod *S3) uploadAllOnTimeout() {
	prod.objectsLock.Lock()
	defer prod.objectsLock.Unlock()
	for s3Path, object := range prod.objects {
		if upload, err := prod.needsUpload(object, 0); err == nil && upload {
			if err := prod.upload(object, true); err == nil {
				object.buffer.CloseAndDelete()
				delete(prod.objects, s3Path)
			}
		}
	}
}

func (prod *S3) uploadAll() error {
	prod.objectsLock.Lock()
	defer prod.objectsLock.Unlock()
	for s3Path, object := range prod.objects {
		if err := prod.upload(object, true); err != nil {
			return err
		}

		object.buffer.CloseAndDelete()
		delete(prod.objects, s3Path)
	}
	return nil
}

func (prod *S3) appendOrUpload(object *objectData, p []byte) error {
	// acquire lock
	object.lock.Lock()
	defer object.lock.Unlock()

	// upload
	needsUpload, err := prod.needsUpload(object, len(p))
	if err != nil {
		return err
	} else if needsUpload {
		if err = prod.upload(object, false); err != nil {
			return err
		}
	}

	// refresh object
	if object.Uploaded {
		buffer, filename, err := prod.newS3Buffer()
		if err != nil {
			return err
		}
		object.buffer.CloseAndDelete()
		object.buffer = buffer
		object.Messages = 0
		object.Compressed = false
		object.Uploaded = false
		object.Filename = filename
		object.Created = time.Now()
	}

	var data []byte
	if object.Messages > 0 {
		data = append(make([]byte, 0), prod.delimiter...)
		data = append(data, p...)
	} else {
		data = p
	}

	// append
	if _, err = object.buffer.Write(data); err != nil {
		Log.Error.Print("S3.appendOrUpload() buffer.Write() error:", err)
		return err
	}
	object.Messages++
	return nil
}

func (prod *S3) transformMessages(messages []core.Message) {
	bufferedMessages := []core.Message{}
	// Format and sort
	for _, msg := range messages {
		msgData, streamID := prod.ProducerBase.Format(msg)

		// Select the correct s3 path
		s3Path, streamMapped := prod.streamMap[streamID]
		if !streamMapped {
			s3Path, streamMapped = prod.streamMap[core.WildcardStreamID]
			if !streamMapped {
				s3Path = core.StreamRegistry.GetStreamName(streamID)
				prod.streamMap[streamID] = s3Path

				shared.Metric.New(s3MetricMessages + s3Path)
				shared.Metric.New(s3MetricMessagesSec + s3Path)
				prod.counters[s3Path] = new(int64)
			}
		}

		// Fetch buffer for this stream
		prod.objectsLock.Lock()
		object, objectExists := prod.objects[s3Path]
		if !objectExists {
			// Create buffer for this s3 path
			buffer, filename, err := prod.newS3Buffer()
			if err != nil {
				prod.Drop(msg)
				prod.objectsLock.Unlock()
				continue
			}
			object = &objectData{
				Compressed: false,
				Created:    time.Now(),
				Filename:   filename,
				Messages:   0,
				S3Path:     s3Path,
				Uploaded:   false,
				buffer:     buffer,
				lock:       new(sync.Mutex),
			}
			prod.objects[s3Path] = object
		}
		prod.objectsLock.Unlock()

		err := prod.appendOrUpload(object, msgData)
		if err != nil {
			prod.Drop(msg)
			continue
		}

		bufferedMessages = append(bufferedMessages, msg)
	}
	// always upload if we aren't using file buffers
	if !prod.useFiles {
		err := prod.uploadAll()
		if prod.closing && err != nil {
			// that was the last chance to upload messages, so drop them
			for _, msg := range bufferedMessages {
				prod.Drop(msg)
			}
		}
	}
}

func (prod *S3) close() {
	prod.closing = true
	defer prod.WorkerDone()
	prod.CloseMessageChannel(prod.bufferMessage)
	prod.batch.Close(prod.transformMessages, prod.GetShutdownTimeout())
	if prod.useFiles && prod.uploadOnShutdown {
		prod.uploadAll()
	}
	prod.storeState()
}

// Produce writes to a buffer that is sent to amazon s3.
func (prod *S3) Produce(workers *sync.WaitGroup) {
	prod.AddMainWorker(workers)

	prod.client = s3.New(session.New(prod.config))
	prod.TickerMessageControlLoop(prod.bufferMessage, prod.flushFrequency, prod.sendBatchOnTimeOut)
}
