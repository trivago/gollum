package producer

import (
	"fmt"
	"gollum/shared"
	"os"
)

// File producer plugin
// Configuration example
//
// - "producer.File":
//   Enable: true
//   File: "/var/log/gollum.log"
//
// File contains the path to the log file to write.
// By default this is set to /var/prod/gollum.log.
type File struct {
	standardProducer
	file *os.File
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

	prod.file, err = os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)

	return prod, nil
}

func (prod File) Produce() {
	defer func() {
		prod.file.Close()
		prod.response <- shared.ProducerControlResponseDone
	}()

	for {
		select {
		case message := <-prod.messages:
			fmt.Fprintln(prod.file, message.Format(prod.forward))

		case command := <-prod.control:
			if command == shared.ProducerControlStop {
				return // ### return, done ###
			}
		default:
			// Don't block
		}
	}
}
