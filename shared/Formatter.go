package shared

const (
	// DefaultTimestamp is the timestamp format string used for messages
	DefaultTimestamp = "2006-01-02 15:04:05 MST"
	// DefaultDelimiter is the default end of message delimiter
	DefaultDelimiter = "\n"
)

// Formatter is the interface definition for message formatters
type Formatter interface {
	// PrepareMessage sets the message to be formatted. This allows the
	// formatter to build up caches for subsequent method calls.
	PrepareMessage(msg Message)

	// GetLength returns the length of a formatted message returned by String()
	// or CopyTo().
	GetLength() int

	// String returns the message as string
	String() string

	// CopyTo copies the message into an existing buffer. It is assumed that
	// dest has enough space to fit GetLength() bytes
	CopyTo(dest []byte) int
}

// Uitob writes an unsigned integer to the end of a given byte buffer.
func Uitob(target []byte, number uint64) {
	for i := len(target) - 1; i >= 0; i-- {
		target[i] = '0' + byte(number%10)
	}
}
