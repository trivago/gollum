package producer

import (
	"fmt"
	"gollum/shared"
	"os"
	"time"
)

// File producer plugin
// Configuration example
//
// - "producer.File":
//   Enable: true
//   File: "/var/log/gollum.log"
//   BatchSize: 4096
//   BatchSizeThreshold: 16777216
//   BatchTimeoutSec: 2
//
// File contains the path to the log file to write.
// By default this is set to /var/prod/gollum.log.
//
// BatchSize defines the number of bytes to be buffered before they are written
// to disk. By default this is set to 8KB.
//
// BatchSizeThreshold defines the maximum number of bytes to buffer before
// messages get dropped. If a message crosses the threshold it is still buffered
// but additional messages will be dropped. By default this is set to 8MB.
//
// BatchTimeoutSec defines the maximum number of seconds to wait after the last
// message arrived before a batch is flushed automatically. By default this is
// set to 5.
type File struct {
	standardProducer
	file               *os.File
	batchSize          int
	batchSizeThreshold int
	batchTimeoutSec    int
}

type fileMessageBuffer struct {
	text        string
	size        int
	lastMessage time.Time
}

func init() {
	shared.Plugin.Register(File{})
}

func (prod File) Create(conf shared.PluginConfig) (shared.Producer, error) {

	err := prod.configureStandardProducer(conf)
	if err != nil {
		return nil, err
	}

	logFile := conf.GetString("File", "/var/prod/gollum.log")
	prod.batchSize = conf.GetInt("BatchSize", 8192)
	prod.batchSizeThreshold = conf.GetInt("BatchThreshold", 8388608)

	prod.file, err = os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)

	return prod, nil
}

func (prod File) writeBatch(batch *fileMessageBuffer) {
	batch.lastMessage = time.Now()

	fmt.Fprintln(prod.file, batch.text)

	batch.size = 0
	batch.text = ""
}

func (prod File) post(batch *fileMessageBuffer, text string) {
	if batch.size < prod.batchSizeThreshold {
		batch.text += text + "\n"
		batch.size += len(text) + 1
		batch.lastMessage = time.Now()

		if batch.size >= prod.batchSize {
			prod.writeBatch(batch)
		}
	}
}

func (prod File) Produce() {
	defer func() {
		prod.file.Close()
		prod.response <- shared.ProducerControlResponseDone
	}()

	batch := fileMessageBuffer{
		text:        "",
		size:        0,
		lastMessage: time.Now(),
	}

	for {
		select {
		case message := <-prod.messages:
			prod.post(&batch, message.Format(prod.forward))

		case command := <-prod.control:
			if command == shared.ProducerControlStop {
				return // ### return, done ###
			}
		default:
			if batch.size > 0 && time.Since(batch.lastMessage).Seconds() > float64(prod.batchTimeoutSec) {
				prod.writeBatch(&batch)
			}
			// Don't block
		}
	}
}
