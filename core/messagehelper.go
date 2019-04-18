package core

import (
	"fmt"

	"github.com/trivago/tgo/tcontainer"
)

// GetDataFunc is a func() to get message content from payload or meta data
// for later handling by plugins
type GetDataFunc func(msg *Message) interface{}

// GetDataAsStringFunc acts as a wrapper around GetDataFunc
// if only string data can be processed.
type GetDataAsStringFunc func(msg *Message) string

// GetDataAsBytesFunc acts as a wrapper around GetDataFunc
// if only []byte] data can be processed.
type GetDataAsBytesFunc func(msg *Message) []byte

// GetMetadataRootFunc acts as a wrapper around a function that returns the
// metadata value as MarshalMap for a fixed key. The function returns an
// error if the value behind the fixed key is not a MarshalMap.
type GetMetadataRootFunc func(msg *Message) (tcontainer.MarshalMap, error)

// ForceMetadataRootFunc works like GetMetadataRootFunc but makes sure that
// the targeted key is existing and usable.
type ForceMetadataRootFunc func(msg *Message) tcontainer.MarshalMap

// SetDataFunc is a func() to store message content to payload or meta data
type SetDataFunc func(msg *Message, content interface{})

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

func getMetadataRoot(msg *Message, root string) (tcontainer.MarshalMap, error) {
	metadata := msg.TryGetMetadata()
	if metadata == nil {
		return nil, fmt.Errorf("no metadata set")
	}

	if len(root) == 0 {
		return metadata, nil
	}

	val, exists := metadata.Value(root)
	if !exists {
		rootValue := tcontainer.MarshalMap{}
		metadata.Set(root, rootValue)
		return rootValue, nil
	}

	return tcontainer.ConvertToMarshalMap(val, nil)
}

func forceMetadataRoot(msg *Message, root string) tcontainer.MarshalMap {
	metadata := msg.GetMetadata()
	if len(root) == 0 {
		return metadata
	}

	val, exists := metadata.Value(root)
	if !exists {
		rootValue := tcontainer.MarshalMap{}
		metadata.Set(root, rootValue)
		return rootValue
	}

	rootValue, err := tcontainer.ConvertToMarshalMap(val, nil)
	if err != nil {
		rootValue = tcontainer.MarshalMap{}
		metadata.Set(root, rootValue)
	}
	return rootValue
}

// NewGetterFor returns a GetDataFunc function
func NewGetterFor(identifier string) GetDataFunc {
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
func NewStringGetterFor(identifier string) GetDataAsStringFunc {
	get := NewGetterFor(identifier)
	return func(msg *Message) string {
		return ConvertToString(get(msg))
	}
}

// NewBytesGetterFor returns a function that gets message content
// as bytes.
func NewBytesGetterFor(identifier string) GetDataAsBytesFunc {
	get := NewGetterFor(identifier)
	return func(msg *Message) []byte {
		return ConvertToBytes(get(msg))
	}
}

// NewMetadataRootGetterFor returns a function that gets a metadata value
// if it is set and if it is a MarshalMap
func NewMetadataRootGetterFor(identifier string) GetMetadataRootFunc {
	return func(msg *Message) (tcontainer.MarshalMap, error) {
		return getMetadataRoot(msg, identifier)
	}
}

// NewForceMetadataRootGetterFor returns a function that always returns a valid
// metadata root. If the key does not exist, it is created. If the key exists
// but is not a MarshalMap, it will be overridden.
func NewForceMetadataRootGetterFor(identifier string) ForceMetadataRootFunc {
	return func(msg *Message) tcontainer.MarshalMap {
		return forceMetadataRoot(msg, identifier)
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

// NewSetterFor returns SetDataFunc function to store message content
func NewSetterFor(identifier string) SetDataFunc {
	if identifier == "" {
		return setPayloadContent
	}

	// we need a lambda to hide away the second parameter
	return func(msg *Message, content interface{}) {
		setMetadataContent(msg, identifier, content)
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
