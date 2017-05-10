// +build integration

package integration

import (
	"github.com/trivago/tgo/ttesting"
	"strings"
	"testing"
)

const (
	TestConfigDistribute = "test_pipeline_distribute.conf"
	TestConfigJoin       = "test_pipeline_join.conf"
	TestConfigDrop       = "test_pipeline_drop.conf"
	TestConfigWildcard   = "test_pipeline_wildcard.conf"
)

func TestPipelineDistribute(t *testing.T) {
	expect := ttesting.NewExpect(t)

	input := []string{"fu"}
	out, err := ExecuteGollum(TestConfigDistribute, input)

	expect.NoError(err)
	expect.Equal("fufu", out.String())
}

func TestPipelineJoin(t *testing.T) {
	expect := ttesting.NewExpect(t)

	input := []string{"fu"}
	out, err := ExecuteGollum(TestConfigJoin, input)

	expect.NoError(err)
	expect.Equal("fufu", out.String())
}

func TestPipelineDrop(t *testing.T) {
	expect := ttesting.NewExpect(t)

	input := []string{"fu"}
	out, err := ExecuteGollum(TestConfigDrop, input)

	expect.NoError(err)
	expect.True(strings.Contains(out.String(), "*"))
	expect.Equal(2, strings.Count(out.String(), "fu"))
}

func TestPipelineWildcard(t *testing.T) {
	expect := ttesting.NewExpect(t)

	input := []string{"fu"}
	out, err := ExecuteGollum(TestConfigWildcard, input)

	expect.NoError(err)
	expect.Equal("fufu", out.String())
}
