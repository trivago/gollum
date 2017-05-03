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
	expect := ttesting.NewExpect(t)

	// execute gollum
	input := []string{"abc", "123", "def"}
	out, err := ExecuteGollum(TestConfigFilter, input, "-ll=2")

	expect.NoError(err)
	expect.True(strings.Contains(out.String(), "(startup)"))

	// get results from file targets
	ResultFileFilterInRouter, err := GetResultFile(TmpTestFilePathFoo)
	expect.NoError(err)

	ResultFileFilterInProducer, err := GetResultFile(TmpTestFilePathBar)
	expect.NoError(err)

	// final expectations filter in router
	expect.True(strings.Contains(ResultFileFilterInRouter.content, "abc"))
	expect.False(strings.Contains(ResultFileFilterInRouter.content, "123"))
	expect.True(strings.Contains(ResultFileFilterInRouter.content, "def"))
	expect.Equal(2, ResultFileFilterInRouter.lines)

	// final expectations filter in producer
	expect.True(strings.Contains(ResultFileFilterInProducer.content, "abc"))
	expect.False(strings.Contains(ResultFileFilterInProducer.content, "123"))
	expect.True(strings.Contains(ResultFileFilterInProducer.content, "def"))
	expect.Equal(2, ResultFileFilterInProducer.lines)
}


