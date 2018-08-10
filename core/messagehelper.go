package core

// GetAppliedContentFunc is a func() to get message content from payload or meta data
// for later handling by plugins
type GetAppliedContentFunc func(msg *Message) interface{}

// SetAppliedContentFunc is a func() to store message content to payload or meta data
type SetAppliedContentFunc func(msg *Message, content interface{})

// GetAppliedContentGetFunction returns a GetAppliedContent function
func GetAppliedContentGetFunction(applyTo string) GetAppliedContentFunc {
	if applyTo != "" {
		return func(msg *Message) interface{} {
			metadata := msg.TryGetMetadata()
			if metadata == nil {
				return []byte{}
			}
			data, exists := metadata.Value(applyTo)
			if exists {
				return data
			}
			return []byte{}
		}
	}

	return func(msg *Message) interface{} {
		return msg.GetPayload()
	}
}

// GetAppliedContentSetFunction returns SetAppliedContent function to store message content
func GetAppliedContentSetFunction(applyTo string) SetAppliedContentFunc {
	if applyTo != "" {
		return func(msg *Message, content interface{}) {
			if content == nil {
				msg.GetMetadata().Delete(applyTo)
			} else {
				msg.GetMetadata().Set(applyTo, content)
			}
		}
	}

	return func(msg *Message, content interface{}) {
		if content == nil {
			msg.ResizePayload(0)
		} else {
			msg.StorePayload(content.([]byte)) // TODO: convert or error?
		}
	}
}
