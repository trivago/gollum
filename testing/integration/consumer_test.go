// +build integration

package integration

import (
	"github.com/trivago/tgo/ttesting"

	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

const (
	TestConfigConsumer = "test_consumer.conf"
)

func TestFileConsumerDefault(t *testing.T) {
	setup()
	expect := ttesting.NewExpect(t)

	// create file
	f, err := os.OpenFile(TmpTestFilePathFoo, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	defer f.Close()
	expect.NoError(err)

	_, err = f.Write([]byte("hello\ngo\n"))
	expect.NoError(err)

	// execute gollum
	cmd := ExecuteGollumAndGetCmd(TestConfigConsumer, []string{}, "-ll=3")
	time.Sleep(2 * time.Second) // wait till gollum should booted

	// write more to file - result for gollum
	_, err = f.Write([]byte("foo\nbar\n"))
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
	expect.Equal(1, ResultFile.lines)
}
