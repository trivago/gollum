package shared

import (
	"fmt"
	"math"
	"strconv"
)

type MessageFormatRLE struct {
	base MessageFormat
}

func CreateMessageFormatRLE(format MessageFormat) MessageFormatRLE {
	return MessageFormatRLE{
		base: format,
	}
}

func (format MessageFormatRLE) GetLength(msg Message) int {
	msgLen := format.base.GetLength(msg)
	if msgLen < 10 {
		return 2 + msgLen
	}

	headerLen := int(math.Log10(float64(msgLen)) + 2)
	return headerLen + msgLen
}

func (format MessageFormatRLE) ToString(msg Message) string {
	return fmt.Sprintf("%d:%s", format.base.GetLength(msg), format.base.ToString(msg))
}

func (format MessageFormatRLE) ToBuffer(msg Message, dest []byte) {
	len := copy(dest, strconv.Itoa(format.base.GetLength(msg)))
	dest[len] = ':'
	format.base.ToBuffer(msg, dest[len+1:])
}
