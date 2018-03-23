// +build integration

package integration

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

const (
	tmpTestFilePathDefault = "/tmp/gollum_test.log"
	tmpTestFilePathFoo     = "/tmp/gollum_test_foo.log"
	tmpTestFilePathBar     = "/tmp/gollum_test_bar.log"
)

var (
	tmpTestFiles = []string{
		tmpTestFilePathDefault,
		tmpTestFilePathFoo,
		tmpTestFilePathBar,
	}
)

func readResultFile(filepath string) (string, int, error) {
	start := time.Now()
	for time.Since(start) < maxFetchResultTime {
		file, err := os.Open(filepath)
		if err != nil {
			time.Sleep(200 * time.Millisecond)
			continue
		}

		defer file.Close()

		// read file
		buffer, err := ioutil.ReadAll(file)
		if err != nil {
			return "", 0, err
		}

		content := string(buffer)
		lines := strings.Count(content, "\n") + 1

		return content, lines, nil
	}

	return "", 0, fmt.Errorf("Timed out while reading from %s", filepath)
}

func removeTestResultFiles() {
	for _, path := range tmpTestFiles {
		if _, err := os.Stat(path); err == nil {
			os.Remove(path)
		}
	}
}
