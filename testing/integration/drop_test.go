// +build integration

package integration

import (
	"github.com/trivago/tgo/ttesting"
	"strings"
	"testing"
)

const (
	TestDropProducer = "test_drop_producer.conf"
	TestDropRouter   = "test_drop_router.conf"
	TestDropConsumer = "test_drop_consumer.conf"
)

func TestDropProducerFilter(t *testing.T) {
	setup()
	expect := ttesting.NewExpect(t)

	// execute gollum
	input := []string{"", "abc", "123", "def"}
	out, err := ExecuteGollum(TestDropProducer, input, "-ll=2")

	expect.NoError(err)
	expect.True(strings.Contains(out.String(), "(startup)"))

	ResultFileDefault, err := getResultFile(tmpTestFilePathDefault)
	expect.NoError(err)

	ResultFileDrops, err := getResultFile(tmpTestFilePathFoo)
	expect.NoError(err)

	// final expectations
	expect.True(strings.Contains(ResultFileDrops.content, "abc"))
	expect.True(strings.Contains(ResultFileDrops.content, "123"))
	expect.True(strings.Contains(ResultFileDrops.content, "def"))
	expect.Equal(1, ResultFileDrops.lines)

	expect.Equal(0, ResultFileDefault.lines)
}

func TestDropRouterFilter(t *testing.T) {
	setup()
	expect := ttesting.NewExpect(t)

	// execute gollum
	input := []string{"abc", "123", "def"}
	out, err := ExecuteGollum(TestDropRouter, input, "-ll=2")

	expect.NoError(err)
	expect.True(strings.Contains(out.String(), "(startup)"))

	ResultFileDefault, err := getResultFile(tmpTestFilePathDefault)
	expect.NoError(err)

	ResultFileDrops, err := getResultFile(tmpTestFilePathFoo)
	expect.NoError(err)

	// final expectations
	expect.True(strings.Contains(ResultFileDrops.content, "abc"))
	expect.False(strings.Contains(ResultFileDrops.content, "123"))
	expect.True(strings.Contains(ResultFileDrops.content, "def"))

	expect.False(strings.Contains(ResultFileDefault.content, "abc"))
	expect.True(strings.Contains(ResultFileDefault.content, "123"))
	expect.False(strings.Contains(ResultFileDefault.content, "def"))
}

func TestDropConsumerFilter(t *testing.T) {
	setup()
	expect := ttesting.NewExpect(t)

	// execute gollum
	input := []string{"abc", "123", "def"}
	out, err := ExecuteGollum(TestDropRouter, input, "-ll=2")

	expect.NoError(err)
	expect.True(strings.Contains(out.String(), "(startup)"))

	ResultFileDefault, err := getResultFile(tmpTestFilePathDefault)
	expect.NoError(err)

	ResultFileDrops, err := getResultFile(tmpTestFilePathFoo)
	expect.NoError(err)

	// final expectations
	expect.True(strings.Contains(ResultFileDrops.content, "abc"))
	expect.False(strings.Contains(ResultFileDrops.content, "123"))
	expect.True(strings.Contains(ResultFileDrops.content, "def"))

	expect.False(strings.Contains(ResultFileDefault.content, "abc"))
	expect.True(strings.Contains(ResultFileDefault.content, "123"))
	expect.False(strings.Contains(ResultFileDefault.content, "def"))
}
