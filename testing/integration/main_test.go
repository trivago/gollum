// +build integration

package integration

import (
	"os"
	"testing"

	"github.com/trivago/gollum/core"
	"github.com/trivago/tgo/ttesting"
)

func TestMain(m *testing.M) {
	removeTestResultFiles()
	defer removeTestResultFiles()
	result := m.Run()
	os.Exit(result)
}

func TestRunableVersion(t *testing.T) {
	expect := ttesting.NewExpect(t)

	cmd, err := StartGollum("", NoStartIndicator, "-v")
	expect.NoError(err)

	err = cmd.Wait()
	expect.NoError(err)

	out := cmd.ReadStdOut()
	expect.Greater(len(out), 0)

	expect.Contains(out, core.GetVersionString())
}

func TestRunableList(t *testing.T) {
	expect := ttesting.NewExpect(t)

	cmd, err := StartGollum("", NoStartIndicator, "-l")
	expect.NoError(err)

	err = cmd.Wait()
	expect.NoError(err)

	out := cmd.ReadStdOut()
	expect.Greater(len(out), 0)

	expect.Contains(out, "consumer")
	expect.Contains(out, "filter")
	expect.Contains(out, "format")
	expect.Contains(out, "producer")
	expect.Contains(out, "router")
}

func TestRunableHelp(t *testing.T) {
	expect := ttesting.NewExpect(t)

	cmd, err := StartGollum("", NoStartIndicator, "-h")
	expect.NoError(err)

	err = cmd.Wait()
	expect.NoError(err)

	out := cmd.ReadStdOut()
	expect.Greater(len(out), 0)

	expect.Contains(out, "-h")
	expect.Contains(out, "help")
	expect.Contains(out, core.GetVersionString())
}
