package producer

import (
	"github.com/trivago/gollum/log"
	"github.com/trivago/gollum/shared"
	"net"
	"strings"
	"sync"
	"time"
)

var fileSocketPrefix = "unix://"

// Socket producer plugin
// Configuration example
//
//   - "producer.Socket":
//     Enable: true
//     Address: "unix:///var/gollum.socket"
//     BufferSizeKB: 4096
//     BufferSizeMaxKB: 16384
//     BatchSizeByte: 4096
//     BatchTimeoutSec: 5
//     Acknowledge: true
//
// Address stores the identifier to connect to.
// This can either be any ip address and port like "localhost:5880" or a file
// like "unix:///var/gollum.socket". By default this is set to ":5880".
//
// BufferSizeKB sets the connection buffer size in KB. By default this is set to
// 1024, i.e. 1 MB buffer.
//
// BufferSizeMaxKB defines the maximum number of bytes to buffer before
// messages get dropped. Any message that crosses the threshold is dropped.
// By default this is set to 8192.
//
// BatchSizeByte defines the number of bytes to be buffered before they are written
// to scribe. By default this is set to 8KB.
//
// BatchTimeoutSec defines the maximum number of seconds to wait after the last
// message arrived before a batch is flushed automatically. By default this is
// set to 5.
//
// Acknowledge can be set to true to expect a response "OK\n" from the server
// after a batch has been sent. This setting is disabled by default.
// If Acknowledge is set to true and a IP-Address is given to Address, TCP is
// used to open the connection, otherwise UDP is used.
type Socket struct {
	shared.ProducerBase
	connection   net.Conn
	batch        *shared.MessageBuffer
	protocol     string
	address      string
	batchSize    int
	batchTimeout time.Duration
	bufferSizeKB int
	runlength    bool
	acknowledge  bool
}

type bufferedConn interface {
	SetWriteBuffer(bytes int) error
}

func init() {
	shared.RuntimeType.Register(Socket{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *Socket) Configure(conf shared.PluginConfig) error {
	err := prod.ProducerBase.Configure(conf)
	if err != nil {
		return err
	}

	bufferSizeMax := conf.GetInt("BufferSizeMaxKB", 8<<10) << 10

	prod.address = conf.GetString("Address", ":5880")
	prod.batchSize = conf.GetInt("BatchSizeByte", 8192)
	prod.batchTimeout = time.Duration(conf.GetInt("BatchTimeoutSec", 5)) * time.Second
	prod.bufferSizeKB = conf.GetInt("BufferSizeKB", 1<<10) // 1 MB
	prod.acknowledge = conf.GetBool("Acknowledge", false)

	if prod.acknowledge {
		prod.protocol = "tcp"
	} else {
		prod.protocol = "udp"
	}

	prod.batch = shared.NewMessageBuffer(bufferSizeMax, prod.Formatter())

	if strings.HasPrefix(prod.address, fileSocketPrefix) {
		prod.address = prod.address[len(fileSocketPrefix):]
		prod.protocol = "unix"
	}

	return nil
}

func (prod *Socket) validate() bool {
	if !prod.acknowledge {
		return true
	}

	response := make([]byte, 2)
	_, err := prod.connection.Read(response)
	if err != nil {
		Log.Error.Print("Socket response error:", err)
		return false
	}
	return string(response) == "OK"
}

func (prod *Socket) sendBatch() {
	// If we have not yet connected or the connection dropped: connect.
	if prod.connection == nil {
		conn, err := net.Dial(prod.protocol, prod.address)

		if err != nil {
			Log.Error.Print("Socket connection error - ", err)
		} else {
			conn.(bufferedConn).SetWriteBuffer(prod.bufferSizeKB << 10)
			prod.connection = conn
		}
	}

	// Flush the buffer to the connection if it is active
	if prod.connection != nil {
		prod.batch.Flush(
			prod.connection,
			prod.validate,
			func(err error) {
				Log.Error.Print("Socket error - ", err)
				prod.connection.Close()
				prod.connection = nil
			})
	}
}

func (prod *Socket) sendBatchOnTimeOut() {
	if prod.batch.ReachedTimeThreshold(prod.batchTimeout) {
		prod.sendBatch()
	}
}

func (prod *Socket) sendMessage(message shared.Message) {
	prod.batch.AppendAndRelease(message)
	if prod.batch.ReachedSizeThreshold(prod.batchSize) {
		prod.sendBatch()
	}
}

func (prod *Socket) flush() {
	for prod.NextNonBlocking(prod.sendMessage) {
	}

	prod.sendBatch()
	prod.batch.WaitForFlush()
}

// Produce writes to a buffer that is sent to a given socket.
func (prod Socket) Produce(threads *sync.WaitGroup) {
	defer func() {
		prod.flush()
		if prod.connection != nil {
			prod.connection.Close()
		}
		prod.MarkAsDone()
	}()

	prod.TickerControlLoop(threads, prod.batchTimeout, prod.sendMessage, nil, prod.sendBatchOnTimeOut)
}
