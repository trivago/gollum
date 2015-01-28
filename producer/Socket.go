package producer

import (
	"fmt"
	"github.com/trivago/gollum/shared"
	"net"
	"strings"
	"time"
)

var fileSocketPrefix = "unix://"

// Console producer plugin
// Configuration example
//
// - "producer.Socket":
//   Enable: true
//   Address: "unix:///var/gollum.socket"
//   BufferSizeKB: 4096
//   BatchSize: 4096
//   BatchSizeThreshold: 16777216
//   BatchTimeoutSec: 5
//
// Address stores the identifier to connect to.
// This can either be any ip address and port like "localhost:5880" or a file
// like "unix:///var/gollum.socket". By default this is set to ":5880".
//
// BufferSize sets the connection buffer size in KB. By default this is set to
// 1024, i.e. 1 MB buffer.
//
// BatchSize defines the number of bytes to be buffered before they are written
// to scribe. By default this is set to 8KB.
//
// BatchSizeThreshold defines the maximum number of bytes to buffer before
// messages get dropped. If a message crosses the threshold it is still buffered
// but additional messages will be dropped. By default this is set to 8MB.
//
// BatchTimeoutSec defines the maximum number of seconds to wait after the last
// message arrived before a batch is flushed automatically. By default this is
// set to 5.
type Socket struct {
	standardProducer
	connection         net.Conn
	protocol           string
	address            string
	batchSize          int
	batchSizeThreshold int
	batchTimeoutSec    int
}

type socketMessageBuffer struct {
	text        string
	size        int
	lastMessage time.Time
}

type bufferedConn interface {
	SetWriteBuffer(bytes int) error
}

func init() {
	shared.Plugin.Register(Socket{})
}

func (prod Socket) Create(conf shared.PluginConfig) (shared.Producer, error) {
	err := prod.configureStandardProducer(conf)
	if err != nil {
		return nil, err
	}

	prod.protocol = "tcp"
	prod.address = conf.GetString("Address", ":5880")
	prod.batchSize = conf.GetInt("BatchSize", 8192)
	prod.batchSizeThreshold = conf.GetInt("BatchSizeThreshold", 8388608)
	prod.batchTimeoutSec = conf.GetInt("BatchTimeoutSec", 5)
	bufferSizeKB := conf.GetInt("BufferSizeKB", 1<<10) // 1 MB

	if strings.HasPrefix(prod.address, fileSocketPrefix) {
		prod.address = prod.address[len(fileSocketPrefix):]
		prod.protocol = "unix"
	}

	prod.connection, err = net.Dial(prod.protocol, prod.address)
	if err != nil {
		shared.Log.Error("Socket connection error:", err)
	} else {
		prod.connection.(bufferedConn).SetWriteBuffer(bufferSizeKB << 10)
	}

	return prod, nil
}

func (prod Socket) sendBatch(batch *socketMessageBuffer) {
	batch.lastMessage = time.Now()

	if prod.connection == nil {
		var err error
		prod.connection, err = net.Dial(prod.protocol, prod.address)

		if err != nil {
			shared.Log.Error("Socket connection error:", err)
		}
	}

	if prod.connection != nil {
		_, err := fmt.Fprintln(prod.connection, batch.text)

		if err != nil {
			shared.Log.Error("Socket error:", err)
			prod.connection.Close()
			prod.connection = nil
		} else {
			batch.text = ""
			batch.size = 0
		}
	}
}

func (prod Socket) post(batch *socketMessageBuffer, text string) {
	if batch.size < prod.batchSizeThreshold {
		batch.text += text + "\n"
		batch.size += len(text) + 1
		batch.lastMessage = time.Now()

		if batch.size >= prod.batchSize {
			prod.sendBatch(batch)
		}
	}
}

func (prod Socket) Produce() {
	defer func() {
		if prod.connection != nil {
			prod.connection.Close()
		}
		prod.response <- shared.ProducerControlResponseDone
	}()

	batch := socketMessageBuffer{
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
				prod.sendBatch(&batch)
			}
			// Don't block
		}
	}
}
