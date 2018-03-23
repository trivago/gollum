// +build integration

package integration

import (
	"testing"
	"time"

	"github.com/trivago/tgo/ttesting"
)

const (
	testRouterConfig = "test_router.conf"
)

func TestDefaultRouter(t *testing.T) {
	removeTestResultFiles()
	expect := ttesting.NewExpect(t)

	// execute gollum
	cmd, err := StartGollum(testRouterConfig, DefaultStartIndicator, "-ll=2")
	expect.NoError(err)

	err = cmd.SendStdIn(time.Second, "abc", "123", "def")
	expect.NoError(err)

	err = cmd.Stop()
	expect.NoError(err)

	// get results from file target
	content, lines, err := readResultFile(tmpTestFilePathDefault)
	expect.NoError(err)

	// final expectations
	expect.Contains(content, "abc")
	expect.Contains(content, "123")
	expect.Contains(content, "def")
	expect.Equal(1, lines)
}

func TestDistributeRouter(t *testing.T) {
	removeTestResultFiles()
	expect := ttesting.NewExpect(t)

	// execute gollum
	cmd, err := StartGollum(testRouterConfig, DefaultStartIndicator, "-ll=2")
	expect.NoError(err)

	err = cmd.SendStdIn(time.Second, "distribute", "router", "test", "456")
	expect.NoError(err)

	err = cmd.Stop()
	expect.NoError(err)

	// get results from file target
	content, lines, err := readResultFile(tmpTestFilePathFoo)
	expect.NoError(err)

	expect.Contains(content, "distribute")
	expect.Contains(content, "routertest")
	expect.Contains(content, "456")
	expect.Equal(1, lines)

	content, lines, err = readResultFile(tmpTestFilePathBar)
	expect.NoError(err)

	expect.Contains(content, "distribute")
	expect.Contains(content, "routertest")
	expect.Contains(content, "456")
	expect.Equal(1, lines)
}
