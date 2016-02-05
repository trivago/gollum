// Copyright 2015-2016 trivago GmbH
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

// +build !windows

package main

import (
	"os"
	"os/signal"
	"syscall"
)

func newSignalHandler() chan os.Signal {
	signalHandler := make(chan os.Signal, 1)
	signal.Notify(signalHandler, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGHUP)
	return signalHandler
}

func translateSignal(sig os.Signal) signalType {
	switch sig {
	case syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1:
		return signalExit

	case syscall.SIGHUP:
		return signalRoll
	}

	return signalNone
}
