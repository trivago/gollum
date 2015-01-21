package consumer

import (
	"bufio"
	"gollum/shared"
	"os"
	"time"
)

type Console struct {
	standardConsumer
}

var ConsoleClassID = shared.Plugin.Register(Console{})

func (cons Console) Create(conf shared.PluginConfig) (shared.Consumer, error) {
	err := cons.configureStandardConsumer(conf)
	return cons, err
}

func (cons Console) readFrom(stream *os.File) {
	var err error
	var message string

	reader := bufio.NewReader(stream)

	for {

		// TODO: This call blocks and prevents this go routine from shutting
		// 		 down properly

		message, err = reader.ReadString('\n')
		if err != nil {
			return // ### return, stream error ###
		}

		for _, stream := range cons.stream {
			postMessage := shared.Message{
				Text:      message[:len(message)-1],
				Stream:    stream,
				Timestamp: time.Now(),
			}

			cons.messages <- postMessage
		}
	}
}

func (cons Console) Consume() {

	go cons.readFrom(os.Stdin)
	defer func() {
		cons.response <- shared.ConsumerControlResponseDone
	}()

	// Wait for control statements

	for {
		command := <-cons.control
		if command == shared.ConsumerControlStop {
			//fmt.Println("Console consumer recieved stop")
			return // ### return ###
		}
	}
}
