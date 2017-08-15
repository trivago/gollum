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
	testConfigConsumer = "test_file_consumer.conf"
)

func TestFileConsumerPoll(t *testing.T) {
	setup()
	expect := ttesting.NewExpect(t)

	resultFile, out, err := executeDefaultTest(tmpTestFilePathFoo, tmpTestFilePathDefault)
	expect.NoError(err)

	// final expectations filter in producer
	expect.Contains(out, "(startup)")

	expect.True(strings.Contains(resultFile.content, "foo"))
	expect.True(strings.Contains(resultFile.content, "bar"))
	expect.Equal(1, resultFile.lines)
}

func TestFileConsumerWatch(t *testing.T) {
	setup()
	expect := ttesting.NewExpect(t)

	resultFile, out, err := executeDefaultTest(tmpTestFilePathBar, tmpTestFilePathDefault)
	expect.NoError(err)

	// final expectations filter in producer
	expect.Contains(out, "(startup)")

	expect.True(strings.Contains(out, "modified file:"))
	expect.True(strings.Contains(out, tmpTestFilePathBar))

	expect.True(strings.Contains(resultFile.content, "foo"))
	expect.True(strings.Contains(resultFile.content, "bar"))
	expect.Equal(1, resultFile.lines)
}

func TestFileConsumerPollWithMove(t *testing.T) {
	setup()
	expect := ttesting.NewExpect(t)

	resultFile, out, err := executeFileRotationTest(tmpTestFilePathFoo, tmpTestFilePathDefault)
	expect.NoError(err)

	// final expectations filter in producer
	expect.Contains(out, "(startup)")

	expect.True(strings.Contains(resultFile.content, "foo"))
	expect.True(strings.Contains(resultFile.content, "bar"))
	expect.True(strings.Contains(resultFile.content, "test"))
	expect.Equal(1, resultFile.lines)
}

// NOTE:
// 	This integration test is now located under testing/integration/noLinux_test.go
//	Reasons are unsupported /unstable file events in fsnotify under linux
/*func TestFileConsumerWatchWithMove(t *testing.T) {
	setup()
	expect := ttesting.NewExpect(t)

	resultFile, out, err := executeFileRotationTest(tmpTestFilePathBar, tmpTestFilePathDefault)
	expect.NoError(err)

	// final expectations filter in producer
	expect.Contains(out.String(), "(startup)")

	expect.True(strings.Contains(resultFile.content, "foo"))
	expect.True(strings.Contains(resultFile.content, "bar"))
	expect.True(strings.Contains(resultFile.content, "test"))
	expect.Equal(1, resultFile.lines)
}*/

func executeDefaultTest(sourceFile string, targetFile string) (resultFile, string, error) {
	// init
	resultFile := resultFile{}
	out := ""

	// create file
	f, err := os.OpenFile(sourceFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	defer f.Close()
	if err != nil {
		return resultFile, out, err
	}

	_, err = f.Write([]byte("hello\ngo\n"))
	if err != nil {
		return resultFile, out, err
	}

	// execute gollum
	cmd := executeGollumAndGetCmd(testConfigConsumer, []string{}, "-ll=3")
	time.Sleep(2 * time.Second) // wait till gollum should booted

	// write more to file - result for gollum
	_, err = f.Write([]byte("foo\nbar\n"))
	if err != nil {
		return resultFile, out, err
	}

	// wait until gollum process is done
	cmd.Wait()

	out = fmt.Sprint(cmd.Stdout)

	// get results from file targets
	resultFile, err = getResultFile(targetFile)
	return resultFile, out, err
}

func executeFileRotationTest(sourceFile string, targetFile string) (resultFile, string, error) {
	// create file
	f, err := os.OpenFile(sourceFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	defer f.Close()
	if err != nil {
		return resultFile{}, "", err
	}

	_, err = f.Write([]byte("hello\ngo\n"))
	if err != nil {
		return resultFile{}, "", err
	}

	// execute gollum
	cmd := executeGollumAndGetCmd(testConfigConsumer, []string{}, "-ll=3")
	time.Sleep(2 * time.Second) // wait till gollum should booted

	// write more to file - result for gollum
	_, err = f.Write([]byte("foo\n"))
	if err != nil {
		return resultFile{}, "", err
	}

	// move file
	f.Close()
	err = os.Rename(sourceFile, sourceFile+".move")
	if err != nil {
		return resultFile{}, "", err
	}

	// create new file
	f, err = os.OpenFile(sourceFile, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return resultFile{}, "", err
	}

	time.Sleep(time.Second * 5)

	// add new content
	_, err = f.Write([]byte("bar\n"))
	if err != nil {
		return resultFile{}, "", err
	}

	_, err = f.Write([]byte("test\n"))
	if err != nil {
		return resultFile{}, "", err
	}

	// wait until gollum process is done
	cmd.Wait()

	out := fmt.Sprint(cmd.Stdout)

	// get results from file targets
	result, err := getResultFile(targetFile)
	if err != nil {
		return resultFile{}, "", err
	}

	// clean up
	os.Remove(sourceFile + ".move")

	return result, out, nil
}
