// +build integration

package integration

import (
	"strings"
	"testing"
	"time"

	"github.com/trivago/tgo/ttesting"
)

const (
	testRegexFilterConfig = "test_regexp_filter.conf"
)

func TestRegexpFilter(t *testing.T) {
	removeTestResultFiles()
	expect := ttesting.NewExpect(t)

	// execute gollum

	cmd, err := StartGollum(testRegexFilterConfig, DefaultStartIndicator, "-ll=2")
	expect.NoError(err)

	err = cmd.SendStdIn(time.Second, "abc", "123", "def")
	expect.NoError(err)

	err = cmd.Stop()
	expect.NoError(err)

	// final expectations filter in router
	content, lines, err := readResultFile(tmpTestFilePathFoo)
	expect.NoError(err)

	expect.Contains(content, "abc")
	expect.False(strings.Contains(content, "123"))
	expect.Contains(content, "def")
	expect.Equal(3, lines)

	// final expectations filter in producer
	content, lines, err = readResultFile(tmpTestFilePathBar)
	expect.NoError(err)

	expect.Contains(content, "abc")
	expect.False(strings.Contains(content, "123"))
	expect.Contains(content, "def")
	expect.Equal(3, lines)

	// final expectations filter in meta data
	content, lines, err = readResultFile(tmpTestFilePathDefault)
	expect.NoError(err)

	expect.Contains(content, "abc")
	expect.Contains(content, "123")
	expect.Contains(content, "def")
	expect.Equal(4, lines)
}
