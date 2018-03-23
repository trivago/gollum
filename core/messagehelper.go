package core

// GetAppliedContent is a func() to get message content from payload or meta data
// for later handling by plugins
type GetAppliedContent func(msg *Message) []byte

// SetAppliedContent is a func() to store message content to payload or meta data
type SetAppliedContent func(msg *Message, content []byte)

// GetAppliedContentGetFunction returns a GetAppliedContent function
func GetAppliedContentGetFunction(applyTo string) GetAppliedContent {
	if applyTo != "" {
		return func(msg *Message) []byte {
			metadata := msg.TryGetMetadata()
			if metadata == nil {
				return []byte{}
			}
			return metadata.GetValue(applyTo)
		}
	}

	return func(msg *Message) []byte {
		return msg.GetPayload()
	}
}

// GetAppliedContentSetFunction returns SetAppliedContent function to store message content
func GetAppliedContentSetFunction(applyTo string) SetAppliedContent {
	if applyTo != "" {
		return func(msg *Message, content []byte) {
			if content == nil {
				msg.GetMetadata().Delete(applyTo)
			} else {
				msg.GetMetadata().SetValue(applyTo, content)
			}
		}
	}

	return func(msg *Message, content []byte) {
		if content == nil {
			msg.ResizePayload(0)
		} else {
			msg.StorePayload(content)
		}
	}
}
