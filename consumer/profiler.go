// Copyright 2015-2017 trivago GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package consumer

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo"
)

// Profiler consumer plugin
//
// The "Profiler" consumer plugin autogenerates messages in user-defined quantity,
// size and density. It can be used to profile producers and configurations and to
// provide a message source for testing.
//
// Before startup, [TemplateCount] template payloads are generated based on the
// format specifier [Message], using characters from [Characters]. The length of
// each template is determined by format size specifier(s) in [Message].
//
// During execution, [Batches] batches of [Runs] messages are generated, with a
// [DelayMs] ms delay between each message. Each message's payload is randomly
// selected from the set of template payloads above.
//
// Configuration example:
//
// # Generate 10 x 10000 messages of 256 bytes
// MyProfiler:
//   Type: "consumer.Profiler"
//   Runs: 10000
//   Batches: 10
//   TemplateCount: 10
//   Characters: "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ01234567890"
//   Message: "%256s"
//	 DelayMs: 0
//   KeepRunning: false
// # Generate a short message every 0.5s, useful for testing and debugging
// JunkGenerator:
//   Type: "consumer.Profiler"
//   Message: "%20s"
//   Streams: "junkstream"
//   Characters: "abcdefghijklmZ"
//   KeepRunning: true
//   Runs: 10000
//   Batches: 3000000
//   DelayMs: 500
//
// Runs defines the number of messages per batch. By default this is set to
// 10000.
//
// Batches defines the number of measurement runs to do. By default this is set
// to 10.
//
// TemplateCount defines the number of message templates to be generated.
// A random message template will be chosen when a message is sent. Templates
// are generated in advance. By default this is set to 10.
//
// Characters defines the characters to be used in generated strings. By default
// these are "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ01234567890 ".
//
// Message defines a go format string to be used to generate the message templates.
// The length of the values generated will be deduced from the format size
// parameter. I.e. "%200d" will generate a digit between 0 and 200, "%10s" will
// generate a string with 10 characters, etc..
// By default this is set to "%256s".
//
// DelayMs defines the number of milliseconds of sleep between messages.
// By default this is set to 0.
//
// KeepRunning can be set to `true` to disable automatic shutdown of gollum after
// profiling is done. This can be used to e.g. read metrics after a profile run.
// By default this is set to `false`.
type Profiler struct {
	core.SimpleConsumer `gollumdoc:"embed_type"`
	profileRuns         int           `config:"Runs" default:"10000"`
	batches             int           `config:"Batches" default:"10"`
	chars               string        `config:"Characters" default:"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ01234567890"`
	message             string        `config:"Message" default:"%256s"`
	delay               time.Duration `config:"DelayMs" default:"0" metric:"ms"`
	keepRunning         bool          `config:"KeepRunning"`
	templates           [][]byte
}

func init() {
	core.TypeRegistry.Register(Profiler{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *Profiler) Configure(conf core.PluginConfigReader) {
	numTemplates := conf.GetInt("TemplateCount", 10)
	cons.templates = make([][]byte, numTemplates)
}

func (cons *Profiler) generateString(size int) string {
	randString := make([]byte, size)
	for i := 0; i < size; i++ {
		randString[i] = cons.chars[rand.Intn(len(cons.chars))]
	}
	return string(randString)
}

func (cons *Profiler) generateTemplate() []byte {
	var dummyValues []interface{}
	messageLen := len(cons.message)

	for idx, char := range cons.message {
		if char == '%' {
			// Get the required length
			size := 0
			searchIdx := idx + 1

			// Get format size
			for searchIdx < messageLen && cons.message[searchIdx] >= '0' && cons.message[searchIdx] <= '9' {
				size = size*10 + int(cons.message[searchIdx]-'0')
				searchIdx++
			}

			// Skip decimal places
			if cons.message[searchIdx] == '.' {
				searchIdx++
				for searchIdx < messageLen && cons.message[searchIdx] >= '0' && cons.message[searchIdx] <= '9' {
					searchIdx++
				}
			}

			// Make sure there is at least 1 number / character
			if size == 0 {
				size = 1
			}

			// Generate values
			switch cons.message[searchIdx] {
			case '%':
				// ignore

			case 'e', 'E', 'f', 'F', 'g', 'G':
				dummyValues = append(dummyValues, rand.Float64()*math.Pow10(size))

			case 'U', 'c':
				dummyValues = append(dummyValues, rune(rand.Intn(1<<16)+32))

			case 'd', 'b', 'o', 'x', 'X':
				dummyValues = append(dummyValues, rand.Intn(int(math.Pow10(size))))

			case 's', 'q', 'v', 'T':
				fallthrough
			default:
				dummyValues = append(dummyValues, cons.generateString(size))
			}
		}
	}

	return []byte(fmt.Sprintf(cons.message, dummyValues...))
}

func (cons *Profiler) profile() {
	defer cons.WorkerDone()

	for i := 0; i < len(cons.templates); i++ {
		cons.templates[i] = cons.generateTemplate()
		cons.Logger.Debugf("Template %d/%d: '%s' (%d bytes)\n",
			i, len(cons.templates), string(cons.templates[i]), len(cons.templates[i]))
	}

	cons.Logger.Debugf("Start profiling with %d templates of %d bytes each",
		len(cons.templates),
		len(cons.templates[0]))

	testStart := time.Now()
	minTime := math.MaxFloat64
	maxTime := 0.0
	messageCount := 0

	for batchIdx := 0; batchIdx < cons.batches && cons.IsActive(); batchIdx++ {
		cons.Logger.Info(fmt.Sprintf("batch %d/%d", batchIdx, cons.batches))
		start := time.Now()

		for i := 0; i < cons.profileRuns && cons.IsActive(); i++ {
			template := cons.templates[rand.Intn(len(cons.templates))]
			messageCount++
			//cons.Logger.Debugf("Enqueuing template: '%s'", template)
			cons.Enqueue(template)

			if cons.delay > 0 && cons.IsActive() {
				time.Sleep(cons.delay)
			}
		}

		runTime := time.Since(start)
		if messageCount%cons.profileRuns == 0 {
			minTime = math.Min(minTime, runTime.Seconds())
			maxTime = math.Max(maxTime, runTime.Seconds())
		}
	}

	runTime := time.Since(testStart)

	cons.Logger.Infof("Overview: %d messages sent in %.4f seconds",
		messageCount,
		runTime.Seconds())

	cons.Logger.Infof("Avg: %4.f msg/sec",
		float64(messageCount)/runTime.Seconds())

	cons.Logger.Infof("Best: %.4f sec = %4.f msg/sec",
		minTime,
		float64(cons.profileRuns)/minTime)

	cons.Logger.Infof("Worst: %.4f sec = %4.f msg/sec",
		maxTime,
		float64(cons.profileRuns)/maxTime)

	if cons.IsActive() {
		cons.Logger.Debug("Profiler done.")
		// Automatically shut down when done
		// TODO: Hack
		if !cons.keepRunning {
			proc, _ := os.FindProcess(os.Getpid())
			proc.Signal(os.Interrupt)
		}
	}
}

// Consume starts a profile run and exits gollum when done
func (cons *Profiler) Consume(workers *sync.WaitGroup) {
	cons.AddMainWorker(workers)

	go tgo.WithRecoverShutdown(cons.profile)
	cons.ControlLoop()
}
