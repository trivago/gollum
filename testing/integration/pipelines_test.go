// +build integration

package integration

import (
	"testing"
	"time"

	"github.com/trivago/tgo/ttesting"
)

const (
	testPipelineDistributeConfig = "test_pipeline_distribute.conf"
	testPipelineJoinConfig       = "test_pipeline_join.conf"
	testPipelineDropConfig       = "test_pipeline_drop.conf"
	testPipelineWildcardConfig   = "test_pipeline_wildcard.conf"
)

func TestPipelineDistribute(t *testing.T) {
	expect := ttesting.NewExpect(t)

	cmd, err := StartGollum(testPipelineDistributeConfig, DefaultStartIndicator, "-ll=2")
	expect.NoError(err)

	err = cmd.SendStdIn(100*time.Millisecond, "### fu")
	expect.NoError(err)

	err = cmd.Stop()
	expect.NoError(err)

	out := cmd.ReadStdOut()
	expect.ContainsN(out, "### fu", 2)
}

func TestPipelineJoin(t *testing.T) {
	expect := ttesting.NewExpect(t)

	cmd, err := StartGollum(testPipelineJoinConfig, DefaultStartIndicator, "-ll=2")
	expect.NoError(err)

	err = cmd.SendStdIn(100*time.Millisecond, "### fu")
	expect.NoError(err)

	err = cmd.Stop()
	expect.NoError(err)

	out := cmd.ReadStdOut()
	expect.ContainsN(out, "### fu", 2)
}

func TestPipelineDrop(t *testing.T) {
	expect := ttesting.NewExpect(t)

	cmd, err := StartGollum(testPipelineDropConfig, DefaultStartIndicator, "-ll=2")
	expect.NoError(err)

	err = cmd.SendStdIn(100*time.Millisecond, "### fu")
	expect.NoError(err)

	err = cmd.Stop()
	expect.NoError(err)

	out := cmd.ReadStdOut()

	expect.ContainsN(out, "STREAM-A", 1)
	expect.ContainsN(out, "STREAM-B", 1)
	expect.ContainsN(out, "### fu", 2)
}

func TestPipelineWildcard(t *testing.T) {
	expect := ttesting.NewExpect(t)

	cmd, err := StartGollum(testPipelineWildcardConfig, DefaultStartIndicator, "-ll=2")
	expect.NoError(err)

	err = cmd.SendStdIn(100*time.Millisecond, "### fu")
	expect.NoError(err)

	err = cmd.Stop()
	expect.NoError(err)

	out := cmd.ReadStdOut()
	expect.ContainsN(out, "### fu", 2)
}
