// +build integration

package integration

import (
	"github.com/trivago/tgo/ttesting"
	"strings"
	"testing"
)

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
