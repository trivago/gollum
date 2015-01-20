package producer

import (
	"fmt"
	"gollum/shared"
	"os"
	"regexp"
)

type Log struct {
	messages chan shared.Message
	control  chan int
	response chan int
	filter   *regexp.Regexp
	file     *os.File
}

var LogClassID = shared.Plugin.Register(Log{})

func (log Log) Create(conf shared.PluginConfig) (shared.Producer, error) {
	var err error

	file, fileSet := conf.Settings["File"]
	filter, filterSet := conf.Settings["Filter"]

	if !filterSet {
		log.filter = nil
	} else {
		log.filter, err = regexp.Compile(filter.(string))
		if err != nil {
			return nil, err
		}
	}

	if !fileSet {
		file = "/var/log/gollum.log"
	}

	log.file, err = os.OpenFile(file.(string), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)

	log.messages = make(chan shared.Message)
	log.control = make(chan int, 1)
	log.response = make(chan int, 1)

	return log, nil
}

func (log Log) Produce() {
	defer func() {
		log.file.Close()
		log.response <- shared.ProducerControlResponseDone
	}()

	for {
		select {
		case message := <-log.messages:
			fmt.Fprintln(log.file, message.Format())

		case command := <-log.control:
			if command == shared.ProducerControlStop {
				fmt.Println("Log producer recieved stop")
				return // ### return, done ###
			}
		default:
			// Don't block
		}
	}
}

func (log Log) Accepts(message shared.Message) bool {
	if log.filter == nil {
		return true // ### return, pass everything ###
	}

	return log.filter.MatchString(message.Text)
}

func (log Log) Control() chan<- int {
	return log.control
}

func (log Log) ControlResponse() <-chan int {
	return log.response
}

func (log Log) Messages() chan<- shared.Message {
	return log.messages
}
