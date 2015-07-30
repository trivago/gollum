// Copyright 2015 trivago GmbH
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
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	"math"
	"math/rand"
	"os"
	"sync"
	"time"
)

// Profiler consumer plugin
// Configuration example
//
//   - "consumer.Profile":
//     Enable: true
//     Runs: 10000
//     Batches: 10
//     TemplateCount: 10
//     Characters: "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ01234567890"
//     Message: "%256s"
//	   DelayMs: 0
//     Stream:
//       - "profile"
//
// The profiler plugin generates Runs x Batches messages and send them to the
// configured streams as fast as possible. This consumer can be used to profile
// producers and/or configurations.
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
// Message defines a go format string to be used to generate the message payloads.
// The length of the values generated will be deducted from the format size
// parameter. I.e. "%200d" will generate a digit between 0 and 200, "%10s" will
// generate a string with 10 characters, etc..
// By default this is set to "%256s".
//
// DelayMs defines the number of milliseconds of sleep between messages.
// By default this is set to 0.
type Profiler struct {
	core.ConsumerBase
	profileRuns int
	batches     int
	templates   [][]byte
	chars       string
	message     string
	quit        bool
	delay       time.Duration
}

var profilerDefaultCharacters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ01234567890 "

func init() {
	shared.RuntimeType.Register(Profiler{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *Profiler) Configure(conf core.PluginConfig) error {
	err := cons.ConsumerBase.Configure(conf)
	if err != nil {
		return err
	}
	numTemplates := conf.GetInt("TemplateCount", 10)

	cons.profileRuns = conf.GetInt("Runs", 10000)
	cons.batches = conf.GetInt("Batches", 10)
	cons.chars = conf.GetString("Characters", profilerDefaultCharacters)
	cons.message = conf.GetString("Message", "%# %256s")
	cons.templates = make([][]byte, numTemplates)
	cons.delay = time.Duration(conf.GetInt("DelayMs", 0)) * time.Millisecond

	return nil
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
	}

	testStart := time.Now()
	minTime := math.MaxFloat64
	maxTime := 0.0
	batchIdx := 0

	for batchIdx = 0; batchIdx < cons.batches && !cons.quit; batchIdx++ {
		Log.Note.Print(fmt.Sprintf("run %d/%d:", batchIdx, cons.batches))
		start := time.Now()

		for i := 0; i < cons.profileRuns && !cons.quit; i++ {
			template := cons.templates[rand.Intn(len(cons.templates))]
			cons.EnqueueCopy(template, uint64(batchIdx*cons.profileRuns+i))
			if cons.delay > 0 {
				time.Sleep(cons.delay)
			}
		}

		runTime := time.Since(start)
		minTime = math.Min(minTime, runTime.Seconds())
		maxTime = math.Max(maxTime, runTime.Seconds())
	}

	runTime := time.Since(testStart)

	Log.Note.Print(fmt.Sprintf(
		"Avg: %.4f sec = %4.f msg/sec",
		runTime.Seconds(),
		float64(cons.profileRuns*batchIdx)/runTime.Seconds()))

	Log.Note.Print(fmt.Sprintf(
		"Best: %.4f sec = %4.f msg/sec",
		minTime,
		float64(cons.profileRuns)/minTime))

	Log.Note.Print(fmt.Sprintf(
		"Worst: %.4f sec = %4.f msg/sec",
		maxTime,
		float64(cons.profileRuns)/maxTime))

	if !cons.quit {
		// Automatically shut down when done
		proc, _ := os.FindProcess(os.Getpid())
		proc.Signal(os.Interrupt)
	}
}

// Consume starts a profile run and exits gollum when done
func (cons *Profiler) Consume(workers *sync.WaitGroup) {
	cons.quit = false
	cons.AddMainWorker(workers)

	go shared.DontPanic(cons.profile)

	defer func() {
		cons.quit = true
	}()

	cons.DefaultControlLoop()
}
