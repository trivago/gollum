package producer

import (
	"fmt"
	"gollum/shared"
	"os"
)

type Log struct {
	standardProducer
	file *os.File
}

var LogClassID = shared.Plugin.Register(Log{})

func (prod Log) Create(conf shared.PluginConfig) (shared.Producer, error) {

	err := prod.configureStandardProducer(conf)
	if err != nil {
		return nil, err
	}

	file, fileSet := conf.Settings["File"]

	if !fileSet {
		file = "/var/prod/gollum.prod"
	}

	prod.file, err = os.OpenFile(file.(string), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)

	return prod, nil
}

func (prod Log) Produce() {
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
				//fmt.Println("prod producer recieved stop")
				return // ### return, done ###
			}
		default:
			// Don't block
		}
	}
}
