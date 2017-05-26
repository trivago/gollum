// +build integration

package integration

import (
	"github.com/trivago/tgo/ttesting"
	"strings"
	"testing"
)

const (
	TestConfigFilter = "test_regexp_filter.conf"
)

func TestRegexpFilter(t *testing.T) {
	setup()
	expect := ttesting.NewExpect(t)

	// execute gollum
	input := []string{"abc", "123", "def"}
	out, err := ExecuteGollum(TestConfigFilter, input, "-ll=2")

	expect.NoError(err)
	expect.True(strings.Contains(out.String(), "(startup)"))

	// final expectations filter in router
	ResultFileFilterInRouter, err := getResultFile(tmpTestFilePathFoo)
	expect.NoError(err)

	expect.True(strings.Contains(ResultFileFilterInRouter.content, "abc"))
	expect.False(strings.Contains(ResultFileFilterInRouter.content, "123"))
	expect.True(strings.Contains(ResultFileFilterInRouter.content, "def"))
	expect.Equal(2, ResultFileFilterInRouter.lines)

	// final expectations filter in producer
	ResultFileFilterInProducer, err := getResultFile(tmpTestFilePathBar)
	expect.NoError(err)

	expect.True(strings.Contains(ResultFileFilterInProducer.content, "abc"))
	expect.False(strings.Contains(ResultFileFilterInProducer.content, "123"))
	expect.True(strings.Contains(ResultFileFilterInProducer.content, "def"))
	expect.Equal(2, ResultFileFilterInProducer.lines)

	// final expectations filter in meta data
	ResultFileFilterInMetadata, err := getResultFile(tmpTestFilePathDefault)
	expect.NoError(err)

	expect.True(strings.Contains(ResultFileFilterInMetadata.content, "abc"))
	expect.True(strings.Contains(ResultFileFilterInMetadata.content, "123"))
	expect.True(strings.Contains(ResultFileFilterInMetadata.content, "def"))
	expect.Equal(3, ResultFileFilterInMetadata.lines)
}
