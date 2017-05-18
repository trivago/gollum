// +build integration

package integration

import (
	"fmt"
	"github.com/trivago/tgo/ttesting"
	"os"
	"strings"
	"testing"
	"time"
)

const (
	testProducerAggregationConfig = "test_aggregated_producers.conf"
	testPipelineAggregationConfig = "test_aggregated_pipeline.conf"
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

func TestProducerAggregationPipeline(t *testing.T) {
	setup()
	expect := ttesting.NewExpect(t)

	// create files
	fileFoo, err := os.OpenFile(TmpTestFilePathFoo, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	defer fileFoo.Close()
	expect.NoError(err)

	_, err = fileFoo.Write([]byte("hello\ngo\n"))
	expect.NoError(err)

	fileBar, err := os.OpenFile(TmpTestFilePathBar, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	defer fileBar.Close()
	expect.NoError(err)

	_, err = fileBar.Write([]byte("hello\ngo\n"))
	expect.NoError(err)

	// execute gollum
	cmd := ExecuteGollumAndGetCmd(testPipelineAggregationConfig, []string{}, "-ll=2")
	time.Sleep(2 * time.Second) // wait till gollum should booted

	// write more to files - result for gollum
	_, err = fileFoo.Write([]byte("foo\n"))
	expect.NoError(err)

	_, err = fileBar.Write([]byte("bar\n"))
	expect.NoError(err)

	// wait until gollum process is done
	cmd.Wait()

	out := fmt.Sprint(cmd.Stdout)
	expect.True(strings.Contains(out, "(startup)"))

	// get results from file targets
	ResultFile, err := GetResultFile(TmpTestFilePathDefault)
	expect.NoError(err)

	// final expectations filter in producer
	expect.True(strings.Contains(ResultFile.content, "foo"))
	expect.True(strings.Contains(ResultFile.content, "bar"))
	expect.Equal(4, ResultFile.lines)
}
