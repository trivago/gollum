package core

import "fmt"

// GetTargetDataFunc is a func() to get message content from payload or meta data
// for later handling by plugins
type GetTargetDataFunc func(msg *Message) interface{}

// GetTargetDataAsStringFunc acts as a wrapper around GetTargetDataFunc
// if only string data can be processed.
type GetTargetDataAsStringFunc func(msg *Message) string

// GetTargetDataAsBytesFunc acts as a wrapper around GetTargetDataFunc
// if only []byte] data can be processed.
type GetTargetDataAsBytesFunc func(msg *Message) []byte

// SetTargetDataFunc is a func() to store message content to payload or meta data
type SetTargetDataFunc func(msg *Message, content interface{})

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

// NewGetterFor returns a GetTargetDataFunc function
func NewGetterFor(identifier string) GetTargetDataFunc {
	if identifier == "" {
		return getPayloadContent
	}

	// we need a lambda to hide away the second parameter
	return func(msg *Message) interface{} {
		return getMetadataContent(msg, identifier)
	}
}

// NewStringGetterFor returns a function that gets message content
// as string.
func NewStringGetterFor(identifier string) GetTargetDataAsStringFunc {
	get := NewGetterFor(identifier)
	return func(msg *Message) string {
		return ConvertToString(get(msg))
	}
}

// NewBytesGetterFor returns a function that gets message content
// as bytes.
func NewBytesGetterFor(identifier string) GetTargetDataAsBytesFunc {
	get := NewGetterFor(identifier)
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

// NewSetterFor returns SetTargetDataFunc function to store message content
func NewSetterFor(applyTo string) SetTargetDataFunc {
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
