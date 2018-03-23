// +build integration

package integration

import (
	"os"
	"testing"
	"time"

	"github.com/trivago/tgo/ttesting"
)

const (
	testProducerAggregationConfig = "test_aggregated_producers.conf"
	testPipelineAggregationConfig = "test_aggregated_pipeline.conf"
)

func TestProducerAggregation(t *testing.T) {
	removeTestResultFiles()
	expect := ttesting.NewExpect(t)

	// execute gollum
	cmd, err := StartGollum(testProducerAggregationConfig, DefaultStartIndicator, "-ll=3")
	expect.NoError(err)

	err = cmd.SendStdIn(time.Second, "abc", "123")
	expect.NoError(err)

	err = cmd.Stop()
	expect.NoError(err)

	content1, _, err := readResultFile(tmpTestFilePathFoo)
	expect.NoError(err)

	content2, _, err := readResultFile(tmpTestFilePathBar)
	expect.NoError(err)

	// final expectations
	expect.Contains(content1, "abc")
	expect.Contains(content1, "abc")

	expect.Contains(content2, "123")
	expect.Contains(content2, "123")
}

func TestProducerAggregationPipeline(t *testing.T) {
	removeTestResultFiles()
	expect := ttesting.NewExpect(t)

	// create files
	fileFoo, err := os.OpenFile(tmpTestFilePathFoo, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	defer fileFoo.Close()
	expect.NoError(err)

	_, err = fileFoo.WriteString("hello\ngo\n")
	expect.NoError(err)

	fileBar, err := os.OpenFile(tmpTestFilePathBar, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	defer fileBar.Close()
	expect.NoError(err)

	_, err = fileBar.WriteString("hello\ngo\n")
	expect.NoError(err)

	// execute gollum
	cmd, err := StartGollum(testPipelineAggregationConfig, DefaultStartIndicator, "-ll=2")
	expect.NoError(err)

	// write more to files - result for gollum
	_, err = fileFoo.WriteString("foo\n")
	expect.NoError(err)

	_, err = fileBar.WriteString("bar\n")
	expect.NoError(err)

	err = cmd.StopAfter(2 * time.Second)
	expect.NoError(err)

	// get results from file targets
	content, lines, err := readResultFile(tmpTestFilePathDefault)
	expect.NoError(err)

	// final expectations filter in producer
	expect.ContainsN(content, "foo", 1)
	expect.ContainsN(content, "bar", 1)
	expect.Equal(5, lines)
}
