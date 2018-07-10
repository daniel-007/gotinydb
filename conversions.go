package gotinydb

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// stringToBytes converter from a string to bytes slice.
// If an error is returned it's has the form of ErrWrongType
func stringToBytes(input interface{}) ([]byte, error) {
	typedInput, ok := input.(string)
	if !ok {
		return nil, ErrWrongType
	}

	lowerCaseString := strings.ToLower(typedInput)

	return []byte(lowerCaseString), nil
}

// intToBytes converter from a int or uint of any size (int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64)
// to bytes slice. If an error is returned it's has the form of ErrWrongType
func intToBytes(input interface{}) ([]byte, error) {
	typedValue := uint64(0)
	switch this := input.(type) {
	case int, int8, int16, int32, int64:
		typedValue = convertIntToAbsoluteUint(input)

	case uint:
		typedValue = uint64(this)
	case uint8:
		typedValue = uint64(this)
	case uint16:
		typedValue = uint64(this)
	case uint32:
		typedValue = uint64(this)
	case uint64:
		typedValue = this
	default:
		return nil, ErrWrongType
	}

	bs := make([]byte, 8)
	binary.BigEndian.PutUint64(bs, typedValue)
	return bs, nil
}

func convertIntToAbsoluteUint(input interface{}) (ret uint64) {
	typedValue := int64(0)

	switch this := input.(type) {
	case int:
		typedValue = int64(this)
	case int8:
		typedValue = int64(this)
	case int16:
		typedValue = int64(this)
	case int32:
		typedValue = int64(this)
	case int64:
		typedValue = int64(this)
	}

	ret = uint64(typedValue) + (math.MaxUint64 / 2) + 1

	return ret
}

func floatToBytes(input interface{}) ([]byte, error) {
	negative := false
	switch this := input.(type) {
	case float32:
		if this < 0 {
			negative = true
		}
	case float64:
		if this < 0 {
			negative = true
		}
	default:
		return nil, ErrWrongType
	}

	floatAsString := fmt.Sprintf("%b", input)
	fmt.Println("floatAsString", floatAsString)

	parts := strings.Split(floatAsString, ".")
	integer, integerParseErr := strconv.ParseInt(parts[0], 10, 64)
	if integerParseErr != nil {
		return nil, integerParseErr
	}
	floating, floatingParseErr := strconv.ParseInt(parts[1], 10, 64)
	if floatingParseErr != nil {
		return nil, floatingParseErr
	}

	fmt.Println("floating", floating)
	if negative {
		floating = -floating
	}
	fmt.Println("floating", floating)

	integerAsBytes, integerToBytesErr := intToBytes(integer)
	if integerToBytesErr != nil {
		return nil, integerToBytesErr
	}
	floatingAsBytes, floatingToBytesErr := intToBytes(floating)
	if floatingToBytesErr != nil {
		return nil, floatingToBytesErr
	}

	return append(integerAsBytes, floatingAsBytes...), nil
}

// timeToBytes converter from a time struct to bytes slice.
// If an error is returned it's has the form of ErrWrongType
func timeToBytes(input interface{}) ([]byte, error) {
	typedInput, ok := input.(time.Time)
	if !ok {
		return nil, ErrWrongType
	}

	return typedInput.MarshalBinary()
}
