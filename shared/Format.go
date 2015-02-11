package shared

const (
	// DefaultTimestamp is the timestamp format string used for messages
	DefaultTimestamp = "2006-01-02 15:04:05 MST"
	// DefaultDelimiter is the default end of message delimiter
	DefaultDelimiter = "\n"
)

// Formatter is the interface definition for message formatters
type Formatter interface {
	// GetLength returns the length of a formatted message returned by String()
	// or CopyTo().
	GetLength(msg Message) int

	// String returns the message as string
	String(msg Message) string

	// CopyTo copies the message into an existing buffer. It is assumed that
	// dest has enough space to fit GetLength() bytes
	CopyTo(dest []byte, msg Message)
}
