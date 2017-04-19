// +build integration

package integration

import (
	"github.com/trivago/tgo/ttesting"
	"strings"
	"testing"
	"os"
)

var TmpTestFiles = []string{TmpTestFilePathDefault, TmpTestFilePathFoo, TmpTestFilePathBar}

func setup() {
	removeTestResultFile()
}

func teardown() {
	removeTestResultFile()
}

func removeTestResultFile() {
	for _, path := range TmpTestFiles {
		if _, err := os.Stat(path); err == nil {
			os.Remove(path)
		}
	}
}

func TestMain(m *testing.M) {
	setup()
	defer teardown() // only called if we panic
	result := m.Run()
	teardown()
	os.Exit(result)
}

func TestRunableVersion(t *testing.T) {
	expect := ttesting.NewExpect(t)
	out, err := ExecuteGollum("", nil, "-v")

	expect.NoError(err)
	expect.True(strings.Contains(out.String(), "Gollum: v"))
}

func TestRunableList(t *testing.T) {
	expect := ttesting.NewExpect(t)
	out, err := ExecuteGollum("", nil, "-l")

	expect.NoError(err)
	expect.True(strings.Contains(out.String(), "consumer"))
	expect.True(strings.Contains(out.String(), "filter"))
	expect.True(strings.Contains(out.String(), "format"))
	expect.True(strings.Contains(out.String(), "producer"))
	expect.True(strings.Contains(out.String(), "router"))
}

func TestRunableHelp(t *testing.T) {
	expect := ttesting.NewExpect(t)
	out, err := ExecuteGollum("", nil, "-h")

	expect.NoError(err)
	expect.True(strings.Contains(out.String(), "Usage: gollum [OPTIONS]"))
	expect.True(strings.Contains(out.String(), "Options:"))
}
