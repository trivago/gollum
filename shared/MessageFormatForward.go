package shared

type MessageFormatForward struct {
}

func CreateMessageFormatForward() MessageFormatForward {
	return MessageFormatForward{}
}

func (format MessageFormatForward) GetLength(msg Message) int {
	return len(msg.Data)
}

func (format MessageFormatForward) ToString(msg Message) string {
	return msg.Data
}

func (format MessageFormatForward) ToBuffer(msg Message, dest []byte) {
	copy(dest, msg.Data)
}
