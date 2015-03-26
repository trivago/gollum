package shared

import "fmt"

type MarshalMap map[string]interface{}

func NewMarshalMap() MarshalMap {
	return make(map[string]interface{})
}

func (mmap MarshalMap) Bool(key string) (bool, error) {
	val, exists := mmap[key]
	if !exists {
		return false, fmt.Errorf(`"%s" is not set`, key)
	}

	boolValue, isBool := val.(bool)
	if !isBool {
		return false, fmt.Errorf(`"%s" is expected to be a boolean`, key)
	}
	return boolValue, nil
}

func (mmap MarshalMap) Int(key string) (int, error) {
	val, exists := mmap[key]
	if !exists {
		return 0, fmt.Errorf(`"%s" is not set`, key)
	}

	intValue, isInt := val.(int)
	if !isInt {
		return 0, fmt.Errorf(`"%s" is expected to be an integer`, key)
	}
	return intValue, nil
}

func (mmap MarshalMap) Uint64(key string) (uint64, error) {
	val, exists := mmap[key]
	if !exists {
		return 0, fmt.Errorf(`"%s" is not set`, key)
	}

	intValue, isInt := val.(uint64)
	if !isInt {
		return 0, fmt.Errorf(`"%s" is expected to be an unsinged integer`, key)
	}
	return intValue, nil
}

func (mmap MarshalMap) String(key string) (string, error) {
	val, exists := mmap[key]
	if !exists {
		return "", fmt.Errorf(`"%s" is not set`, key)
	}

	strValue, isString := val.(string)
	if !isString {
		return "", fmt.Errorf(`"%s" is expected to be a string`, key)
	}
	return strValue, nil
}

func (mmap MarshalMap) StringArray(key string) ([]interface{}, error) {
	val, exists := mmap[key]
	if !exists {
		return nil, fmt.Errorf(`"%s" is not set`, key)
	}

	switch val.(type) {
	case string:
		return []interface{}{val.(string)}, nil
	default:
		return mmap.Array(key)
	}
}

func (mmap MarshalMap) Array(key string) ([]interface{}, error) {
	val, exists := mmap[key]
	if !exists {
		return nil, fmt.Errorf(`"%s" is not set`, key)
	}

	arrayValue, isArray := val.([]interface{})
	if !isArray {
		return nil, fmt.Errorf(`"%s" is expected to be an array`, key)
	}
	return arrayValue, nil
}

func (mmap MarshalMap) Map(key string) (map[interface{}]interface{}, error) {
	val, exists := mmap[key]
	if !exists {
		return nil, fmt.Errorf(`"%s" is not set`, key)
	}

	mapValue, isMap := val.(map[interface{}]interface{})
	if !isMap {
		return nil, fmt.Errorf(`"%s" is expected to be a map`, key)
	}
	return mapValue, nil
}

func (mmap MarshalMap) MarshalMap(key string) (MarshalMap, error) {
	interfaceMap, err := mmap.Map(key)
	if err != nil {
		return nil, err
	}

	result := NewMarshalMap()
	for key, value := range interfaceMap {
		stringKey, isString := key.(string)
		if !isString {
			return nil, fmt.Errorf(`"%s" is expected to be a map[string]interface{}. Key is not a string`, key)
		}
		result[stringKey] = value
	}

	return result, nil
}
