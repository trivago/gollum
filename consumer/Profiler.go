package consumer

import (
	"fmt"
	"gollum/shared"
	"time"
)

type Profiler struct {
	standardConsumer
	profileRuns int
	batches     int
}

func init() {
	shared.Plugin.Register(Profiler{})
}

func (cons Profiler) Create(conf shared.PluginConfig) (shared.Consumer, error) {
	err := cons.configureStandardConsumer(conf)

	cons.profileRuns = conf.GetInt("Runs", 10000)
	cons.batches = conf.GetInt("Batches", 10)

	return cons, err
}

func (cons Profiler) profile() {
	for b := 0; b < cons.batches; b++ {
		start := time.Now()
		for i := 0; i < cons.profileRuns; i++ {
			cons.postMessage(fmt.Sprintf("%d/%d", i, cons.profileRuns))
		}
		runTime := time.Since(start)

		shared.Log.Note(fmt.Sprintf(
			"Profile run #%d: %.4f sec = %4.f msg/sec\n",
			b, runTime.Seconds(),
			float64(cons.profileRuns)/runTime.Seconds()))
	}
}

func (cons Profiler) Consume() {
	go cons.profile()

	defer func() {
		cons.response <- shared.ConsumerControlResponseDone
	}()

	// Wait for control statements

	for {
		command := <-cons.control
		if command == shared.ConsumerControlStop {
			return // ### return ###
		}
	}
}
