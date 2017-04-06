// +build integration

package integration

import (
	"bytes"
	"github.com/trivago/tgo/ttesting"
	"io"
	"os"
	"strings"
	"testing"
)

const (
	TestConfigFileName = "test_regexp_filter.conf"
	TmpTestFilePath    = "/tmp/gollum_test.log"
)

func TestMain(m *testing.M) {

	defer teardown() // only called if we panic
	result := m.Run()
	teardown()
	os.Exit(result)
}

func teardown() {
	if _, err := os.Stat(TmpTestFilePath); err == nil {
		os.Remove(TmpTestFilePath)
	}
}

func TestRegexpFilter(t *testing.T) {
	expect := ttesting.NewExpect(t)

	// execute gollum
	input := []string{"abc", "123", "def"}
	out, err := ExecuteGollum(TestConfigFileName, input, "-ll=2")

	expect.NoError(err)
	expect.True(strings.Contains(out.String(), "(startup)"))

	// get results from file target
	fileContent, err := GetFileContentAsString(TmpTestFilePath)
	expect.NoError(err)

	// final expectations
	expect.True(strings.Contains(fileContent, "abc"))
	expect.False(strings.Contains(fileContent, "123"))
	expect.True(strings.Contains(fileContent, "def"))

	file, _ := os.Open(TmpTestFilePath)
	lines, _ := lineCounter(file)
	expect.Equal(2, lines)
}

func lineCounter(r io.Reader) (int, error) {
	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := r.Read(buf)
		count += bytes.Count(buf[:c], lineSep)

		switch {
		case err == io.EOF:
			return count, nil

		case err != nil:
			return count, err
		}
	}
}
