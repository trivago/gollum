package core

const APPLY_TO_PAYLOAD = "payload"

// GetAppliedContent is a func() to get message content from payload or meta data
// for later handling by plugins
type GetAppliedContent func(msg *Message) []byte

// GetAppliedContentFunction returns a GetAppliedContent function
func GetAppliedContentFunction(applyTo string) GetAppliedContent {
	if applyTo != "" && applyTo != APPLY_TO_PAYLOAD {
		return func(msg *Message) []byte {
			return msg.MetaData().GetValue(applyTo)
		}
	}

	return func(msg *Message) []byte {
		return msg.Data()
	}
}
