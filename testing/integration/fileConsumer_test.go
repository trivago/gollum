// +build integration

package integration

import (
	"github.com/trivago/tgo/ttesting"

	"os"
	"testing"
	"time"
)

const (
	testFileConsumerConfig     = "test_file_consumer.conf"
	testFileConsumerWhite      = "test_file_consumer_white.conf"
	testFileConsumerBlack      = "test_file_consumer_black.conf"
	testFileConsumerBlackWhite = "test_file_consumer_blackwhite.conf"
	testFileConsumerWait       = 3 * time.Second
)

func TestFileConsumerPoll(t *testing.T) {
	removeTestResultFiles()
	expect := ttesting.NewExpect(t)

	testFileChange(expect, tmpTestFilePathFoo)
}

func TestFileConsumerWatch(t *testing.T) {
	removeTestResultFiles()
	expect := ttesting.NewExpect(t)

	testFileChange(expect, tmpTestFilePathBar)
}

func TestFileConsumerPollWithMove(t *testing.T) {
	removeTestResultFiles()
	expect := ttesting.NewExpect(t)

	testFileChangeAndMove(expect, tmpTestFilePathFoo)
}

func TestFileConsumerWhite(t *testing.T) {
	removeTestResultFiles()
	expect := ttesting.NewExpect(t)

	testGlob(expect, testFileConsumerWhite)
}

func TestFileConsumerBlack(t *testing.T) {
	removeTestResultFiles()
	expect := ttesting.NewExpect(t)

	testGlob(expect, testFileConsumerBlack)
}

func TestFileConsumerBlackWhite(t *testing.T) {
	removeTestResultFiles()
	expect := ttesting.NewExpect(t)

	testGlob(expect, testFileConsumerBlackWhite)
}

// helper functions

func generateTestFile(name, content string) (*os.File, error) {
	file, err := os.OpenFile(name, os.O_APPEND|os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}

	_, err = file.Write([]byte(content))
	return file, err
}

func testFileChange(expect ttesting.Expect, sourceFile string) {
	file, err := generateTestFile(tmpTestFilePathFoo, "test\n")
	expect.NoError(err)

	cmd, err := StartGollum(testFileConsumerConfig, DefaultStartIndicator, "-ll=2")
	expect.NoError(err)

	_, err = file.WriteString("foo\nbar\n")
	expect.NoError(err)

	err = cmd.StopAfter(time.Second)
	expect.NoError(err)
	file.Close()

	// Check results

	content, lines, err := readResultFile(tmpTestFilePathDefault)
	expect.NoError(err)

	expect.Contains(content, "foo")
	expect.Contains(content, "bar")
	expect.Equal(1, lines)
}

func testFileChangeAndMove(expect ttesting.Expect, sourceFile string) {
	file, err := generateTestFile(tmpTestFilePathFoo, "test\n")
	expect.NoError(err)

	cmd, err := StartGollum(testFileConsumerConfig, DefaultStartIndicator, "-ll=2")
	expect.NoError(err)

	_, err = file.WriteString("foo\n")
	expect.NoError(err)
	file.Close()

	// Move file and create a new one

	err = os.Rename(sourceFile, sourceFile+".move")
	expect.NoError(err)

	file, err = generateTestFile(tmpTestFilePathFoo, "bar\n")
	expect.NoError(err)

	_, err = file.WriteString("baz\n")
	expect.NoError(err)

	// Stop and clean

	err = cmd.StopAfter(time.Second)
	expect.NoError(err)

	file.Close()
	os.Remove(sourceFile + ".move")

	// Check results

	content, lines, err := readResultFile(tmpTestFilePathDefault)
	expect.NoError(err)

	expect.Contains(content, "foo")
	expect.Contains(content, "bar")
	expect.Contains(content, "baz")
	expect.Equal(1, lines)
}

func testGlob(expect ttesting.Expect, config string) {
	file, err := generateTestFile(tmpTestFilePathGlob0, "0\n")
	expect.NoError(err)
	file.Close()

	file, err = generateTestFile(tmpTestFilePathGlob1, "1\n")
	expect.NoError(err)
	file.Close()

	file, err = generateTestFile(tmpTestFilePathGlob2, "2\n")
	expect.NoError(err)
	file.Close()

	cmd, err := StartGollum(config, DefaultStartIndicator, "-ll=2")
	expect.NoError(err)

	err = cmd.StopAfter(time.Second)
	expect.NoError(err)

	// Check results

	content, _, err := readResultFile(tmpTestFilePathDefault)
	expect.NoError(err)

	expect.Contains(content, "0")
	expect.Contains(content, "1")
	expect.Equal(2, len(content))
}
