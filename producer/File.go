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

var FileClassID = shared.Plugin.Register(File{})

func (prod File) Create(conf shared.PluginConfig) (shared.Producer, error) {

	err := prod.configureStandardProducer(conf)
	if err != nil {
		return nil, err
	}

	file, fileSet := conf.Settings["File"]
	if !fileSet {
		file = "/var/prod/gollum.log"
	}

	prod.file, err = os.OpenFile(file.(string), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)

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
			fmt.Fprintln(prod.file, message.Format())

		case command := <-prod.control:
			if command == shared.ProducerControlStop {
				return // ### return, done ###
			}
		default:
			// Don't block
		}
	}
}
