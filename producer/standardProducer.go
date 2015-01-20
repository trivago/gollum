package producer

import (
	"gollum/shared"
	"regexp"
)

type standardProducer struct {
	messages chan shared.Message
	control  chan int
	response chan int
	filter   *regexp.Regexp
}

func (prod *standardProducer) configureStandardProducer(conf shared.PluginConfig) error {
	var err error
	filter, filterSet := conf.Settings["Filter"]

	if !filterSet {
		prod.filter = nil
	} else {
		prod.filter, err = regexp.Compile(filter.(string))
		if err != nil {
			return err
		}
	}

	prod.messages = make(chan shared.Message)
	prod.control = make(chan int, 1)
	prod.response = make(chan int, 1)

	return nil
}

func (prod standardProducer) Accepts(message shared.Message) bool {
	if prod.filter == nil {
		return true // ### return, pass everything ###
	}

	return prod.filter.MatchString(message.Text)
}

func (prod standardProducer) Control() chan<- int {
	return prod.control
}

func (prod standardProducer) ControlResponse() <-chan int {
	return prod.response
}

func (prod standardProducer) Messages() chan<- shared.Message {
	return prod.messages
}
