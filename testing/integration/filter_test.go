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

	// get results from file target
	ResultFile, err := GetResultFile(TmpTestFilePathDefault)
	expect.NoError(err)

	// final expectations
	expect.True(strings.Contains(ResultFile.content, "abc"))
	expect.False(strings.Contains(ResultFile.content, "123"))
	expect.True(strings.Contains(ResultFile.content, "def"))
	expect.Equal(2, ResultFile.lines)
}


