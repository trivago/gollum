package consumer

import (
	"fmt"
	"github.com/trivago/gollum/log"
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
// - "consumer.Console":
//   Enable: true
//
// This consumer does not define any options beside the standard ones.
type Profiler struct {
	standardConsumer
	profileRuns int
	batches     int
	length      int
	quit        bool
}

func init() {
	shared.RuntimeType.Register(Profiler{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *Profiler) Configure(conf shared.PluginConfig) error {
	err := cons.standardConsumer.Configure(conf)
	if err != nil {
		return err
	}

	cons.profileRuns = conf.GetInt("Runs", 10000)
	cons.batches = conf.GetInt("Batches", 10)
	cons.length = conf.GetInt("Length", 256)

	return nil
}

var stringBase = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ01234567890 _.!?/&%$§'")

func (cons *Profiler) profile() {

	randString := make([]rune, cons.length)
	for i := 0; i < cons.length; i++ {
		randString[i] = stringBase[rand.Intn(len(stringBase))]
	}

	testStart := time.Now()

	var msg string
	minTime := math.MaxFloat64
	maxTime := 0.0

	for b := 0; b < cons.batches; b++ {

		start := time.Now()
		for i := 0; i < cons.profileRuns; i++ {
			msg = fmt.Sprintf("%d/%d %s", i, cons.profileRuns, string(randString))
			cons.postMessage(msg)

			if cons.quit {
				cons.markAsDone()
				return
			}
		}

		runTime := time.Since(start)
		minTime = math.Min(minTime, runTime.Seconds())
		maxTime = math.Max(maxTime, runTime.Seconds())

		Log.Note.Print(fmt.Sprintf(
			"Profile run #%d: %.4f sec = %4.f msg/sec",
			b, runTime.Seconds(),
			float64(cons.profileRuns)/runTime.Seconds()))
	}

	runTime := time.Since(testStart)

	Log.Note.Print(fmt.Sprintf(
		"Avg: %.4f sec = %4.f msg/sec",
		runTime.Seconds(),
		float64(cons.profileRuns*cons.batches)/runTime.Seconds()))

	Log.Note.Print(fmt.Sprintf(
		"Best: %.4f sec = %4.f msg/sec",
		minTime,
		float64(cons.profileRuns)/minTime))

	Log.Note.Print(fmt.Sprintf(
		"Worst: %.4f sec = %4.f msg/sec",
		maxTime,
		float64(cons.profileRuns)/maxTime))

	cons.markAsDone()

	proc, _ := os.FindProcess(os.Getpid())
	proc.Signal(os.Interrupt)
}

// Consume starts a profile run and exits gollum when done
func (cons Profiler) Consume(threads *sync.WaitGroup) {
	go cons.profile()
	defer func() {
		cons.quit = true
	}()

	cons.defaultControlLoop(threads)
}
