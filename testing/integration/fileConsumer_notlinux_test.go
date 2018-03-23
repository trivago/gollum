// +build integration,!linux

package integration

import (
	"github.com/trivago/tgo/ttesting"

	"testing"
)

// NOTE:
// 	This integration test is normally located under testing/integration/fileConsumer_test.go
//	Reasons are unsupported /unstable file events in fsnotify under linux
func TestFileConsumerWatchWithMove(t *testing.T) {
	removeTestResultFiles()
	expect := ttesting.NewExpect(t)

	testFileChangeAndMove(expect, tmpTestFilePathFoo)
}
