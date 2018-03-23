// +build integration

package integration

import (
	"strings"
	"testing"
	"time"

	"github.com/trivago/tgo/ttesting"
)

const (
	testDropProducerConfig = "test_drop_producer.conf"
	testDropRouterConfig   = "test_drop_router.conf"
	testDropConsumerConfig = "test_drop_consumer.conf"
)

func TestDropProducerFilter(t *testing.T) {
	removeTestResultFiles()
	expect := ttesting.NewExpect(t)

	// execute gollum
	cmd, err := StartGollum(testDropProducerConfig, DefaultStartIndicator, "-ll=2")
	expect.NoError(err)

	err = cmd.SendStdIn(time.Second, "abc", "123", "def")
	expect.NoError(err)

	err = cmd.Stop()
	expect.NoError(err)

	content, lines, err := readResultFile(tmpTestFilePathFoo)
	expect.NoError(err)

	expect.Contains(content, "123")
	expect.False(strings.Contains(content, "abc"))
	expect.False(strings.Contains(content, "def"))
	expect.Contains(content, "changed")

	content, lines, err = readResultFile(tmpTestFilePathDefault)
	expect.NoError(err)

	expect.Contains(content, "abc")
	expect.Contains(content, "def")
	expect.False(strings.Contains(content, "123"))
	expect.Equal(1, lines)
}

func TestDropRouterFilter(t *testing.T) {
	removeTestResultFiles()
	expect := ttesting.NewExpect(t)

	// execute gollum
	cmd, err := StartGollum(testDropRouterConfig, DefaultStartIndicator, "-ll=2")
	expect.NoError(err)

	err = cmd.SendStdIn(time.Second, "abc", "123", "def")
	expect.NoError(err)

	err = cmd.Stop()
	expect.NoError(err)

	content, _, err := readResultFile(tmpTestFilePathFoo)
	expect.NoError(err)

	expect.Contains(content, "123")
	expect.False(strings.Contains(content, "abc"))
	expect.False(strings.Contains(content, "def"))

	content, _, err = readResultFile(tmpTestFilePathDefault)
	expect.NoError(err)

	expect.Contains(content, "abc")
	expect.Contains(content, "def")
	expect.False(strings.Contains(content, "123"))
}

func TestDropConsumerFilter(t *testing.T) {
	removeTestResultFiles()
	expect := ttesting.NewExpect(t)

	// execute gollum
	cmd, err := StartGollum(testDropConsumerConfig, DefaultStartIndicator, "-ll=2")
	expect.NoError(err)

	err = cmd.SendStdIn(time.Second, "abc", "123", "def")
	expect.NoError(err)

	err = cmd.Stop()
	expect.NoError(err)

	content, _, err := readResultFile(tmpTestFilePathFoo)
	expect.NoError(err)

	expect.Contains(content, "123")
	expect.False(strings.Contains(content, "abc"))
	expect.False(strings.Contains(content, "def"))
	expect.Contains(content, "changed")

	content, _, err = readResultFile(tmpTestFilePathDefault)
	expect.NoError(err)

	expect.Contains(content, "abc")
	expect.Contains(content, "def")
	expect.False(strings.Contains(content, "123"))
}
