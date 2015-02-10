package consumer

import (
	"github.com/trivago/gollum/log"
	"github.com/trivago/gollum/shared"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
)

const (
	fileBufferGrowSize = 1024
)

// File consumer plugin
// Configuration example
//
// - "consumer.File":
//   Enable: true
//   File: "test.txt"
//	 Delimiter: "\n"
//
// File is a mandatory setting and contains the file to read. The file will be
// read from beginning to end and the reader will stay attached until the
// consumer is stopped. This means appends to the file will be recognized by
// gollum.
//
// Delimiter defines the end of a message inside the file. By default this is
// set to "\n".
type File struct {
	standardConsumer
	file      *os.File
	fileName  string
	delimiter string
	quit      bool
}

func init() {
	shared.RuntimeType.Register(File{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *File) Configure(conf shared.PluginConfig) error {
	err := cons.standardConsumer.Configure(conf)
	if err != nil {
		return err
	}

	if !conf.HasValue("File") {
		panic("No file configured for consumer.File")
	}

	escapeChars := strings.NewReplacer("\\n", "\n", "\\r", "\r", "\\t", "\t")

	cons.fileName = conf.GetString("File", "")
	cons.delimiter = escapeChars.Replace(conf.GetString("Delimiter", "\n"))
	cons.quit = false

	return nil
}

func (cons *File) readFrom(stream io.Reader, threads *sync.WaitGroup) {
	buffer := shared.CreateBufferedReader(fileBufferGrowSize, cons.postMessageFromSlice)

	for !cons.quit {
		err := buffer.Read(cons.file, cons.delimiter)

		if err != nil && err != io.EOF && !cons.quit {
			Log.Error.Print("Error reading stdin: ", err)
			return
		}

		if err == io.EOF {
			runtime.Gosched()
		}
	}

	cons.markAsDone()
}

// Consume listens to stdin.
func (cons File) Consume(threads *sync.WaitGroup) {
	var err error
	cons.file, err = os.OpenFile(cons.fileName, os.O_RDONLY, 0666)
	if err != nil {
		Log.Error.Print("File open error:", err)
		return
	}

	cons.quit = false
	go cons.readFrom(cons.file, threads)
	defer func() {
		cons.quit = true
		cons.file.Close()
	}()

	cons.defaultControlLoop(threads)
}
