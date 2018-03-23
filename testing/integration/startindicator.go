// +build integration

package integration

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type StartIndicator interface {
	HasStarted(handle *GollumHandle) error
}

type StartIndicatorFunc func(handle *GollumHandle) error

type FileStartIndicatorData struct {
	name string
}

func (f StartIndicatorFunc) HasStarted(handle *GollumHandle) error {
	return f(handle)
}

var (
	DefaultStartIndicator = StartIndicatorFunc(defaultStartIndicatorFunc)
	NoStartIndicator      = StartIndicatorFunc(noStartIndicatorFunc)
)

func defaultStartIndicatorFunc(handle *GollumHandle) error {
	const startupString = "(startup)"
	start := time.Now()

	for !strings.Contains(handle.ReadStdOut(), startupString) {
		if time.Since(start) > maxStartupWaitTime {
			fmt.Println(handle.ReadStdOut())
			return fmt.Errorf("Timed out waiting for string \"%s\"", startupString)
		}
		time.Sleep(200 * time.Millisecond)
	}

	return nil
}

func noStartIndicatorFunc(handle *GollumHandle) error {
	return nil
}

func FileStartIndicator(name string) FileStartIndicatorData {
	return FileStartIndicatorData{
		name: name,
	}
}

func (f FileStartIndicatorData) HasStarted(handle *GollumHandle) error {
	start := time.Now()

	for time.Since(start) < maxStartupWaitTime {
		_, err := os.Stat(f.name)
		if err == nil {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}

	return fmt.Errorf("Timed out waiting for file %s", f.name)
}
