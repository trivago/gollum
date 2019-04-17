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
	tmpTestFilePathGlob0   = "/tmp/gollum_test_glob0.log"
	tmpTestFilePathGlob1   = "/tmp/gollum_test_glob1.log"
	tmpTestFilePathGlob2   = "/tmp/gollum_test_glob2.log"
)

var (
	tmpTestFiles = []string{
		tmpTestFilePathDefault,
		tmpTestFilePathFoo,
		tmpTestFilePathBar,
		tmpTestFilePathGlob0,
		tmpTestFilePathGlob1,
		tmpTestFilePathGlob2,
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

	return "", 0, fmt.Errorf("timed out while reading from %s", filepath)
}

func removeTestResultFiles() {
	for _, path := range tmpTestFiles {
		if _, err := os.Stat(path); err == nil {
			os.Remove(path)
		}
	}
}
