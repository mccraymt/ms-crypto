package gas

import (
	"errors"
	"strconv"
)

// Interfaces => type for handling []interface{} conversions
type Interfaces []interface{}

// Ints => type for handling []int conversions
type Ints []int

// Strings => type for handling []string conversions
type Strings []string

// ToInts => converts Interface Slice to Int Slice
func (slice Interfaces) ToInts() ([]int, error) {
	var ints = make([]int, len(slice))
	for i, v := range slice {
		switch t := v.(type) {
		case int:
			ints[i] = v.(int)
		case float64:
			ints[i] = int(v.(float64))
		case string:
			if p, err := strconv.Atoi(v.(string)); err == nil {
				ints[i] = p
			} else {
				return nil, err
			}
		default:
			return nil, errors.New("unmapped type: " + t.(string))
		}
	}
	return ints, nil
}

// ToStrings => converts Interface Slice to String Slice
func (slice Interfaces) ToStrings() ([]string, error) {
	var strings = make([]string, len(slice))
	for i, v := range slice {
		switch t := v.(type) {
		case string:
			strings[i] = v.(string)
		case int:
			strings[i] = strconv.Itoa(v.(int))
		case bool:
			strings[i] = strconv.FormatBool(v.(bool))
		case int32:
			strings[i] = strconv.FormatInt(int64(v.(int32)), 10)
		case int64:
			strings[i] = strconv.FormatInt(v.(int64), 10)
		case float32:
			strings[i] = strconv.FormatFloat(float64(v.(float32)), 'E', -1, 32)
		case float64:
			strings[i] = strconv.FormatFloat(v.(float64), 'E', -1, 64)
		default:
			return nil, errors.New("unmapped type: " + t.(string))
		}
	}
	return strings, nil
}

// ToInterfaces => converts Int Slice to Interface Slice
func (slice Ints) ToInterfaces() ([]interface{}, error) {
	var interfaces = make([]interface{}, len(slice))
	for i, v := range slice {
		interfaces[i] = v
	}
	return interfaces, nil
}

// ToInterfaces => converts Int Slice to Interface Slice
func (slice Strings) ToInterfaces() ([]interface{}, error) {
	var interfaces = make([]interface{}, len(slice))
	for i, v := range slice {
		interfaces[i] = v
	}
	return interfaces, nil
}
