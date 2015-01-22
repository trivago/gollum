package shared

import (
	"fmt"
	"time"
)

// Internal message exchange format struct.
type Message struct {
	Text      string
	Stream    string
	Timestamp time.Time
}

// Create a message in the default format.
func (msg Message) Format(forward bool) string {
	if forward {
		return msg.Text
	}

	return fmt.Sprintf("%s | %s | %s",
		msg.GetDateString(),
		msg.Stream,
		msg.Text)
}

// Format the date in the default format
func (msg Message) GetDateString() string {
	return fmt.Sprintf("%s", msg.Timestamp.Format("2006-01-02 15:04:05 MST"))
}
