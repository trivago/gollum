package trivago

import (
	"github.com/trivago/gollum/shared"
	"testing"
)

func TestZap(t *testing.T) {
	expect := shared.NewExpect(t)
	testData := []byte("\n\"")
	zapInvalidChars(testData)

	expect.BytesEq(testData, []byte("\t'"))
}
