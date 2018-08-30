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

func getPayloadContent(msg *Message) interface{} {
	return msg.GetPayload()
}

func getMetadataContent(msg *Message, key string) interface{} {
	metadata := msg.TryGetMetadata()
	if metadata == nil {
		return []byte{}
	}
	if data, ok := metadata.Value(key); ok {
		return data
	}
	return []byte{}
}

// NewGetAppliedContentFunc returns a GetAppliedContentFunc function
func NewGetAppliedContentFunc(applyTo string) GetAppliedContentFunc {
	if applyTo == "" {
		return getPayloadContent
	}

	// we need a lambda to hide away the second parameter
	return func(msg *Message) interface{} {
		return getMetadataContent(msg, applyTo)
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

func setMetadataContent(msg *Message, key string, content interface{}) {
	if content == nil {
		msg.GetMetadata().Delete(key)
	} else {
		msg.GetMetadata().Set(key, content)
	}
}

func setPayloadContent(msg *Message, content interface{}) {
	if content == nil {
			msg.data.payload = msg.data.payload[:0]
	} else {
		msg.StorePayload(ConvertToBytes(content))
	}
}

// NewSetAppliedContentFunc returns SetAppliedContentFunc function to store message content
func NewSetAppliedContentFunc(applyTo string) SetAppliedContentFunc {
	if applyTo == "" {
		return setPayloadContent
	}

	// we need a lambda to hide away the second parameter
	return func(msg *Message, content interface{}) {
		setMetadataContent(msg, applyTo, content)
	}
}

// ConvertToBytes tries to covert data into a byte string.
// String and []byte types will be converted directly, all other types
// are converted via Sprintf("%v").
func ConvertToBytes(val interface{}) []byte {
	if bytes, isBytes := val.([]byte); isBytes {
		return bytes
	}

	if str, isString := val.(string); isString {
		return []byte(str)
	}

	return []byte(fmt.Sprintf("%v", val))
}

// ConvertToString tries to covert data into a string.
// String and []byte types will be converted directly, all other types
// are converted via Sprintf("%v").
func ConvertToString(val interface{}) string {
	if bytes, isBytes := val.([]byte); isBytes {
		return string(bytes)
	}

	if str, isString := val.(string); isString {
		return str
	}

	return fmt.Sprintf("%v", val)
}
