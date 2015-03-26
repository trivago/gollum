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

func (mmap MarshalMap) Float64(key string) (float64, error) {
	val, exists := mmap[key]
	if !exists {
		return 0, fmt.Errorf(`"%s" is not set`, key)
	}

	floatValue, isFloat := val.(float64)
	if !isFloat {
		return 0, fmt.Errorf(`"%s" is expected to be a float64`, key)
	}
	return floatValue, nil
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

func castToStringArray(key string, value interface{}) ([]string, error) {
	switch value.(type) {
	case string:
		return []string{value.(string)}, nil

	case []interface{}:
		arrayVal := value.([]interface{})
		stringArray := make([]string, 0, len(arrayVal))

		for _, val := range arrayVal {
			strValue, isString := val.(string)
			if !isString {
				return nil, fmt.Errorf(`"%s" does not contain string keys`, key)
			}
			stringArray = append(stringArray, strValue)
		}
		return stringArray, nil

	case []string:
		return value.([]string), nil

	default:
		return nil, fmt.Errorf(`"%s" is not a valid string array type`, key)
	}
}

func (mmap MarshalMap) StringArray(key string) ([]string, error) {
	val, exists := mmap[key]
	if !exists {
		return nil, fmt.Errorf(`"%s" is not set`, key)
	}

	return castToStringArray(key, val)
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

func (mmap MarshalMap) StringMap(key string) (map[string]string, error) {
	val, exists := mmap[key]
	if !exists {
		return nil, fmt.Errorf(`"%s" is not set`, key)
	}

	switch val.(type) {
	case map[interface{}]interface{}:
		result := make(map[string]string)
		interfaceMap, _ := val.(map[interface{}]interface{})

		for key, value := range interfaceMap {
			stringKey, isString := key.(string)
			if !isString {
				return nil, fmt.Errorf(`"%s" is expected to be a map[string]string. Key is not a string`, key)
			}
			stringValue, isString := value.(string)
			if !isString {
				return nil, fmt.Errorf(`"%s" is expected to be a map[string]string. Value is not a string`, key)
			}
			result[stringKey] = stringValue
		}
		return result, nil

	case map[string]interface{}:
		result := make(map[string]string)
		interfaceMap, _ := val.(map[string]interface{})

		for key, value := range interfaceMap {
			stringValue, isString := value.(string)
			if !isString {
				return nil, fmt.Errorf(`"%s" is expected to be a map[string]string. Value is not a string`, key)
			}
			result[key] = stringValue
		}
		return result, nil

	case map[string]string:
		return val.(map[string]string), nil

	default:
		return nil, fmt.Errorf(`"%s" is expected to be a map[string]string`, key)
	}
}

func (mmap MarshalMap) StringArrayMap(key string) (map[string][]string, error) {
	val, exists := mmap[key]
	if !exists {
		return nil, fmt.Errorf(`"%s" is not set`, key)
	}

	switch val.(type) {
	case map[interface{}][]interface{}:
		interfaceMap := val.(map[interface{}][]interface{})
		result := make(map[string][]string)

		for key, value := range interfaceMap {
			strKey, isString := key.(string)
			if !isString {
				return nil, fmt.Errorf(`"%s" is expected to be a map[string][]string. Key is not a string`, key)
			}
			arrayValue, err := castToStringArray(strKey, value)
			if err != nil {
				return nil, fmt.Errorf(`"%s" is expected to be a map[string][]string. Value is not a []string`, key)
			}
			result[strKey] = arrayValue
		}
		return result, nil

	case map[interface{}]interface{}:
		interfaceMap := val.(map[interface{}]interface{})
		result := make(map[string][]string)

		for key, value := range interfaceMap {
			strKey, isString := key.(string)
			if !isString {
				return nil, fmt.Errorf(`"%s" is expected to be a map[string][]string. Key is not a string`, key)
			}
			arrayValue, err := castToStringArray(strKey, value)
			if err != nil {
				return nil, fmt.Errorf(`"%s" is expected to be a map[string][]string. Value is not a []string`, key)
			}
			result[strKey] = arrayValue
		}
		return result, nil

	case map[string]interface{}:
		interfaceMap := val.(map[string]interface{})
		result := make(map[string][]string)

		for key, value := range interfaceMap {
			arrayValue, err := castToStringArray(key, value)
			if err != nil {
				return nil, fmt.Errorf(`"%s" is expected to be a map[string][]string. Value is not a []string`, key)
			}
			result[key] = arrayValue
		}
		return result, nil

	case map[string][]string:
		return val.(map[string][]string), nil

	default:
		return nil, fmt.Errorf(`"%s" is expected to be a map[string][]string`, key)
	}
}

func (mmap MarshalMap) MarshalMap(key string) (MarshalMap, error) {
	val, exists := mmap[key]
	if !exists {
		return nil, fmt.Errorf(`"%s" is not set`, key)
	}

	switch val.(type) {
	case map[interface{}]interface{}:
		result := NewMarshalMap()
		interfaceMap, _ := val.(map[interface{}]interface{})

		for key, value := range interfaceMap {
			stringKey, isString := key.(string)
			if !isString {
				return nil, fmt.Errorf(`"%s" is expected to be a map[string]interface{}. Key is not a string`, key)
			}
			result[stringKey] = value
		}
		return result, nil

	case map[string]interface{}:
		return val.(map[string]interface{}), nil

	case MarshalMap:
		return val.(MarshalMap), nil

	default:
		return nil, fmt.Errorf(`"%s" is expected to be a map[string]interface{}`, key)
	}
}
