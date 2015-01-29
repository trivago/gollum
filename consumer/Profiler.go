package consumer

import (
	"fmt"
	"gollum/shared"
	"math"
	"math/rand"
	"os"
	"sync"
	"time"
)

type Profiler struct {
	standardConsumer
	profileRuns int
	batches     int
	length      int
}

func init() {
	shared.Plugin.Register(Profiler{})
}

func (cons Profiler) Create(conf shared.PluginConfig, pool *shared.BytePool) (shared.Consumer, error) {
	err := cons.configureStandardConsumer(conf, pool)

	cons.profileRuns = conf.GetInt("Runs", 10000)
	cons.batches = conf.GetInt("Batches", 10)
	cons.length = conf.GetInt("Length", 256)

	return cons, err
}

var stringBase = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ01234567890 _.!?/&%$§'")

func (cons Profiler) profile() {

	randString := make([]rune, cons.length)
	for i := 0; i < cons.length; i++ {
		randString[i] = stringBase[rand.Intn(len(stringBase))]
	}

	testStart := time.Now()

	var msg string
	minTime := math.MaxFloat64
	for b := 0; b < cons.batches; b++ {

		start := time.Now()
		for i := 0; i < cons.profileRuns; i++ {
			msg = fmt.Sprintf("%d/%d %s", i, cons.profileRuns, string(randString))
			cons.postMessage(msg)
		}

		runTime := time.Since(start)
		minTime = math.Min(minTime, runTime.Seconds())

		shared.Log.Note(fmt.Sprintf(
			"Profile run #%d: %.4f sec = %4.f msg/sec",
			b, runTime.Seconds(),
			float64(cons.profileRuns)/runTime.Seconds()))
	}

	runTime := time.Since(testStart)
	shared.Log.Note(fmt.Sprintf(
		"Total: %.4f sec = %4.f msg/sec",
		runTime.Seconds(),
		float64(cons.profileRuns*cons.batches)/runTime.Seconds()))

	shared.Log.Note(fmt.Sprintf(
		"Best: %.4f sec = %4.f msg/sec",
		minTime,
		float64(cons.profileRuns)/minTime))

	proc, _ := os.FindProcess(os.Getpid())
	proc.Signal(os.Interrupt)
}

func (cons Profiler) Consume(threads *sync.WaitGroup) {
	go cons.profile()

	// Wait for control statements

	for {
		command := <-cons.control
		if command == shared.ConsumerControlStop {
			return // ### return ###
		}
	}
}
