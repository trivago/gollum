package shared

import (
	"fmt"
)

type MessageFormatSimple struct {
	delimiter string
}

func CreateMessageFormatSimple(delimiter string) MessageFormatSimple {
	return MessageFormatSimple{
		delimiter: delimiter,
	}
}

func (format MessageFormatSimple) GetLength(msg Message) int {
	return len(msg.Data) + len(format.delimiter)
}

func (format MessageFormatSimple) ToString(msg Message) string {
	return fmt.Sprintf("%s%s", msg.Data, format.delimiter)
}

func (format MessageFormatSimple) ToBuffer(msg Message, dest []byte) {
	len := copy(dest, format.delimiter)
	len += copy(dest[len:], msg.Data)
}
