// +build integration

package integration

import (
	"github.com/trivago/tgo/ttesting"
	"strings"
	"testing"
)

const (
	testProducerAggregationConfig = "test_aggregated_producers.conf"
)

func TestProducerAggregation(t *testing.T) {
	setup()
	expect := ttesting.NewExpect(t)

	// execute gollum
	input := []string{"", "abc", "123"}
	out, err := ExecuteGollum(testProducerAggregationConfig, input, "-ll=2")

	expect.NoError(err)
	expect.True(strings.Contains(out.String(), "(startup)"))

	resultFileProducer1, err := GetResultFile(TmpTestFilePathFoo)
	expect.NoError(err)

	resultFileProducer2, err := GetResultFile(TmpTestFilePathBar)
	expect.NoError(err)

	// final expectations
	expect.True(strings.Contains(resultFileProducer1.content, "abc"))
	expect.True(strings.Contains(resultFileProducer2.content, "abc"))

	expect.True(strings.Contains(resultFileProducer1.content, "123"))
	expect.True(strings.Contains(resultFileProducer2.content, "123"))
}