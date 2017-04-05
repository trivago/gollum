// +build integration

package integration

import (
	"github.com/trivago/tgo/ttesting"
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
	input := []string{"abc", "123"}
	out, err := ExecuteGollum(TestConfigFileName, input, "-ll=2")

	expect.NoError(err)
	expect.True(strings.Contains(out.String(), "(startup)"))

	// get results from file target
	fileContent, err := GetFileContentAsString(TmpTestFilePath)
	expect.NoError(err)

	// final expectations
	expect.True(strings.Contains(fileContent, "abc"))
	expect.False(strings.Contains(fileContent, "123"))
}
