package shared

import (
	"bytes"
	"runtime"
	"testing"
)

// Expect is a helper construct for unittesting
type Expect struct {
	t *testing.T
}

// NewExpect returns a new Expect struct bound to a unittest object
func NewExpect(t *testing.T) Expect {
	return Expect{t}
}

func (e Expect) print(message string) {
	_, file, line, _ := runtime.Caller(1)
	e.t.Errorf("%s(%d): %s", file, line, message)
}

func (e Expect) printf(format string, args ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	e.t.Errorf("%s(%d): "+format, file, line, args)
}

// True tests if the given value is true
func (e Expect) True(val bool) bool {
	if !val {
		e.print("Expected true")
		return false
	}
	return true
}

// False tests if the given value is false
func (e Expect) False(val bool) bool {
	if val {
		e.print("Expected false")
		return false
	}
	return true
}

// IntEq tests two integers on equality
func (e Expect) IntEq(val1, val2 int) bool {
	if val1 != val2 {
		e.printf("Expected \"%d\", got \"%d\"", val1, val2)
		return false
	}
	return true
}

// StringEq tests two strings on equality
func (e Expect) StringEq(val1, val2 string) bool {
	if val1 != val2 {
		e.printf("Expected \"%s\", got \"%s\"", val1, val2)
		return false
	}
	return true
}

// BytesEq tests two byte slices on equality
func (e Expect) BytesEq(val1, val2 []byte) bool {
	if !bytes.Equal(val1, val2) {
		e.printf("Expected %v, got %v", val1, val2)
		return false
	}
	return true
}

// MapSetEq tests if a key/value pair is set in a given map and if value is of
// an expected value
func (e Expect) MapSetEq(data map[string]interface{}, key string, value string) bool {
	val, valSet := data[key]
	if !valSet {
		e.printf("Expected key \"%s\" not found", key)
		return false
	}
	if val != value {
		e.printf("Expected \"%s\" for \"%s\" got \"%s\"", value, key, val)
		return false
	}
	return true
}

// MapSet tests if a given key exists in a map
func (e Expect) MapSet(data map[string]interface{}, key string) bool {
	_, valSet := data[key]
	if !valSet {
		e.printf("Expected key \"%s\" to be set", key)
		return false
	}
	return true
}

// MapNotSet tests if a given key does not exist in a map
func (e Expect) MapNotSet(data map[string]interface{}, key string) bool {
	_, valSet := data[key]
	if valSet {
		e.printf("Expected key \"%s\" not to be set", key)
		return false
	}
	return true
}
