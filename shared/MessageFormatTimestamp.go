package shared

import "fmt"

const (
	messageFormatTimestampSeparator = " | "
)

type MessageFormatTimestamp struct {
	timestampFormat string
	delimiter       string
}

func CreateMessageFormatTimestamp(timestampFormat string, delimiter string) MessageFormatTimestamp {
	return MessageFormatTimestamp{
		timestampFormat: timestampFormat,
		delimiter:       delimiter,
	}
}

func (format MessageFormatTimestamp) GetLength(msg Message) int {
	return len(format.timestampFormat) + len(messageFormatTimestampSeparator) + len(msg.Data) + len(format.delimiter)
}

func (format MessageFormatTimestamp) ToString(msg Message) string {
	return fmt.Sprintf("%s%s%s%s", msg.Timestamp.Format(format.timestampFormat), messageFormatTimestampSeparator, msg.Data, format.delimiter)
}

func (format MessageFormatTimestamp) ToBuffer(msg Message, dest []byte) {
	len := copy(dest[:], msg.Timestamp.Format(format.timestampFormat))
	len += copy(dest[len:], messageFormatTimestampSeparator)
	len += copy(dest[len:], msg.Data)
	len += copy(dest[len:], format.delimiter)
}
