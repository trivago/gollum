// +build integration
// +build !linux

package integration

import (
	"github.com/trivago/tgo/ttesting"

	"strings"
	"testing"
)

// NOTE:
// 	This integration test is normally located under testing/integration/fileConsumer_test.go
//	Reasons are unsupported /unstable file events in fsnotify under linux
func TestFileConsumerWatchWithMove(t *testing.T) {
	setup()
	expect := ttesting.NewExpect(t)

	resultFile, out, err := executeFileRotationTest(tmpTestFilePathBar, tmpTestFilePathDefault)
	expect.NoError(err)

	// final expectations filter in producer
	expect.True(strings.Contains(out, "(startup)"))

	expect.True(strings.Contains(resultFile.content, "foo"))
	expect.True(strings.Contains(resultFile.content, "bar"))
	expect.True(strings.Contains(resultFile.content, "test"))
	expect.Equal(1, resultFile.lines)
}
