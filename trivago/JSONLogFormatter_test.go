package trivago

import (
	"bytes"
	"testing"
)

func TestZap(t *testing.T) {
	testData := []byte("\n\"")
	zapInvalidChars(testData)

	if !bytes.Equal(testData, []byte("\t'")) {
		t.Error("zapInvalidChars produced unexpected result.")
	}
}
