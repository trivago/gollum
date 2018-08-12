package core

import "fmt"

// GetAppliedContentFunc is a func() to get message content from payload or meta data
// for later handling by plugins
type GetAppliedContentFunc func(msg *Message) interface{}

// GetAppliedContentAsStringFunc acts as a wrapper around GetAppliedContentFunc
// if only string data can be processed.
type GetAppliedContentAsStringFunc func(msg *Message) string

// GetAppliedContentAsBytesFunc acts as a wrapper around GetAppliedContentFunc
// if only []byte] data can be processed.
type GetAppliedContentAsBytesFunc func(msg *Message) []byte

// SetAppliedContentFunc is a func() to store message content to payload or meta data
type SetAppliedContentFunc func(msg *Message, content interface{})

// NewGetAppliedContentFunc returns a GetAppliedContentFunc function
func NewGetAppliedContentFunc(applyTo string) GetAppliedContentFunc {
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

// NewGetAppliedContentAsStringFunc returns a function that gets message content
// as string.
func NewGetAppliedContentAsStringFunc(applyTo string) GetAppliedContentAsStringFunc {
	get := NewGetAppliedContentFunc(applyTo)
	return func(msg *Message) string {
		return ConvertToString(get(msg))
	}
}

// NewGetAppliedContentAsBytesFunc returns a function that gets message content
// as bytes.
func NewGetAppliedContentAsBytesFunc(applyTo string) GetAppliedContentAsBytesFunc {
	get := NewGetAppliedContentFunc(applyTo)
	return func(msg *Message) []byte {
		return ConvertToBytes(get(msg))
	}
}

// NewSetAppliedContentFunc returns SetAppliedContentFunc function to store message content
func NewSetAppliedContentFunc(applyTo string) SetAppliedContentFunc {
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
			msg.StorePayload(ConvertToBytes(content))
		}
	}
}

// ConvertToBytes tries to covert data into a byte string.
// String and []byte types will be converted directly, all other types
// are converted via Sprintf("%v").
func ConvertToBytes(val interface{}) []byte {
	switch val.(type) {
	case string:
		return []byte(val.(string))

	case []byte:
		return val.([]byte)

	default:
		return []byte(fmt.Sprintf("%v", val))
	}
}

// ConvertToString tries to covert data into a string.
// String and []byte types will be converted directly, all other types
// are converted via Sprintf("%v").
func ConvertToString(val interface{}) string {
	switch val.(type) {
	case string:
		return val.(string)

	case []byte:
		return string(val.([]byte))

	default:
		return fmt.Sprintf("%v", val)
	}
}
