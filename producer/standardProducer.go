package producer

import (
	"github.com/trivago/gollum/log"
	"github.com/trivago/gollum/shared"
	"regexp"
	"sync"
	"time"
)

// Producer base class
// All producers support a common subset of configuration options:
//
// - "producer.Something":
//   Enable: true
//   Channel: 1024
//   Formatter: "format.Timestamp"
//   Stream:
//      - "error"
//      - "default"
//
// Enable switches the consumer on or off. By default this value is set to true.
//
// Channel sets the size of the channel used to communicate messages. By default
// this value is set to 1024.
//
// Stream contains either a single string or a list of strings defining the
// message channels this producer will consume. By default this is set to "*"
// which means "all streams".
//
// Fromatter sets a formatter to use. Each formatter has its own set of options
// which can be set here, too. By default this is set to format.Forward
type standardProducer struct {
	messages chan shared.Message
	control  chan shared.ProducerControl
	filter   *regexp.Regexp
	state    *shared.PluginRunState
	format   shared.Formatter
}

type producerError struct {
	message string
}

func (err producerError) Error() string {
	return err.message
}

func (prod *standardProducer) Configure(conf shared.PluginConfig) error {
	prod.messages = make(chan shared.Message, conf.Channel)
	prod.control = make(chan shared.ProducerControl, 1)
	prod.filter = nil
	prod.state = new(shared.PluginRunState)

	plugin, err := shared.RuntimeType.NewPlugin(conf.GetString("Formatter", "format.Forward"), conf)
	if err != nil {
		return err
	}

	prod.format = plugin.(shared.Formatter)

	filter := conf.GetString("Filter", "")

	if filter != "" {
		prod.filter, err = regexp.Compile(filter)
		if err != nil {
			Log.Error.Print("Regex error: ", err)
		}
	}

	return nil
}

func (prod *standardProducer) markAsActive(threads *sync.WaitGroup) {
	prod.state.WaitGroup = threads
	prod.state.WaitGroup.Add(1)
	prod.state.Active = true
}

func (prod standardProducer) markAsDone() {
	prod.state.WaitGroup.Done()
	prod.state.Active = false
}

func (prod standardProducer) IsActive() bool {
	return prod.state.Active
}

func (prod standardProducer) Accepts(message shared.Message) bool {
	if prod.filter == nil {
		return true // ### return, pass everything ###
	}

	return prod.filter.MatchString(message.Data)
}

func (prod standardProducer) Control() chan<- shared.ProducerControl {
	return prod.control
}

func (prod standardProducer) Messages() chan<- shared.Message {
	return prod.messages
}

func (prod standardProducer) processCommand(command shared.ProducerControl, onRoll func()) bool {
	switch command {
	default:
		// Do nothing
	case shared.ProducerControlStop:
		return true // ### return ###
	case shared.ProducerControlRoll:
		if onRoll != nil {
			onRoll()
		}
	}

	return false
}

func (prod standardProducer) defaultControlLoop(threads *sync.WaitGroup, onMessage func(msg shared.Message), onRoll func()) {
	prod.markAsActive(threads)

	for prod.IsActive() {
		select {
		case message := <-prod.messages:
			onMessage(message)

		case command := <-prod.control:
			if prod.processCommand(command, onRoll) {
				return // ### return ###
			}
		}
	}
}

func (prod standardProducer) tickerControlLoop(threads *sync.WaitGroup, timeOutSec int, onMessage func(msg shared.Message), onTimeOut func(), onRoll func()) {
	flushTicker := time.NewTicker(time.Duration(timeOutSec) * time.Second)
	prod.markAsActive(threads)

	for prod.IsActive() {
		select {
		case message := <-prod.messages:
			onMessage(message)

		case command := <-prod.control:
			if prod.processCommand(command, onRoll) {
				return // ### return ###
			}

		case <-flushTicker.C:
			onTimeOut()
		}
	}
}
