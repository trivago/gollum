// +build integration

package integration

import (
	"github.com/trivago/tgo/ttesting"
	"strings"
	"testing"
)

func TestRunable(t *testing.T) {
	expect := ttesting.NewExpect(t)
	out, err := ExecuteGollum("", nil, "-v")

	expect.Nil(err)
	expect.True(strings.Contains(out.String(), "Gollum: v"))
}
