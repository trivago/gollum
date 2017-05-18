// +build integration

package integration

import (
	"github.com/trivago/tgo/ttesting"
	"strings"
	"testing"
)

const (
	TestConfigRouter = "test_router.conf"
)

func TestDefaultRouter(t *testing.T) {
	setup()
	expect := ttesting.NewExpect(t)

	// execute gollum
	input := []string{"abc", "123", "def"}
	out, err := ExecuteGollum(TestConfigRouter, input, "-ll=2")

	expect.NoError(err)
	expect.True(strings.Contains(out.String(), "(startup)"))

	// get results from file target
	ResultFile, err := getResultFile(tmpTestFilePathDefault)
	expect.NoError(err)

	// final expectations
	expect.True(strings.Contains(ResultFile.content, "abc"))
	expect.True(strings.Contains(ResultFile.content, "123"))
	expect.True(strings.Contains(ResultFile.content, "def"))
	expect.Equal(1, ResultFile.lines)
}

func TestDistributeRouter(t *testing.T) {
	setup()
	expect := ttesting.NewExpect(t)

	// execute gollum
	input := []string{"distribute", "router", "test", "456"}
	out, err := ExecuteGollum(TestConfigRouter, input, "-ll=2")

	expect.NoError(err)
	expect.True(strings.Contains(out.String(), "(startup)"))

	// get results from file target
	ResultFileFoo, err := getResultFile(tmpTestFilePathFoo)
	expect.NoError(err)

	ResultFileBar, err := getResultFile(tmpTestFilePathBar)
	expect.NoError(err)

	// final expectations
	expect.True(strings.Contains(ResultFileFoo.content, "distribute"))
	expect.True(strings.Contains(ResultFileFoo.content, "routertest"))
	expect.True(strings.Contains(ResultFileFoo.content, "456"))
	expect.Equal(1, ResultFileFoo.lines)

	expect.True(strings.Contains(ResultFileBar.content, "distribute"))
	expect.True(strings.Contains(ResultFileBar.content, "routertest"))
	expect.True(strings.Contains(ResultFileBar.content, "456"))
	expect.Equal(1, ResultFileBar.lines)
}
