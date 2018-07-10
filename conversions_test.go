package gotinydb

import (
	"bytes"
	"math"
	"reflect"
	"testing"
	"time"
)

func TestStringConversion(t *testing.T) {
	if _, err := stringToBytes("string to convert"); err != nil {
		t.Error(err)
		return
	}

	if _, err := stringToBytes(time.Now()); err == nil {
		t.Error(err)
		return
	}
}

func TestIntConversion(t *testing.T) {
	if _, err := intToBytes(31497863415); err != nil {
		t.Error(err)
		return
	}
	if _, err := intToBytes(int8(-117)); err != nil {
		t.Error(err)
		return
	}
	if _, err := intToBytes(int16(3847)); err != nil {
		t.Error(err)
		return
	}
	if _, err := intToBytes(int32(-7842245)); err != nil {
		t.Error(err)
		return
	}
	if _, err := intToBytes(int64(22416315751)); err != nil {
		t.Error(err)
		return
	}
	if _, err := intToBytes(uint(31497863415)); err != nil {
		t.Error(err)
		return
	}
	if _, err := intToBytes(uint8(117)); err != nil {
		t.Error(err)
		return
	}
	if _, err := intToBytes(uint16(3847)); err != nil {
		t.Error(err)
		return
	}
	if _, err := intToBytes(uint32(7842245)); err != nil {
		t.Error(err)
		return
	}
	if _, err := intToBytes(uint64(22416315751)); err != nil {
		t.Error(err)
		return
	}
}

func TestIntOrdering(t *testing.T) {
	neg, _ := intToBytes(-1)
	null, _ := intToBytes(0)
	pos, _ := intToBytes(1)

	if !reflect.DeepEqual(neg, []byte{127, 255, 255, 255, 255, 255, 255, 255}) {
		t.Errorf("negative values is not what is expected: \n%v\n%v", neg, []byte{127, 255, 255, 255, 255, 255, 255, 255})
	} else if !reflect.DeepEqual(null, []byte{128, 0, 0, 0, 0, 0, 0, 0}) {
		t.Errorf("null values is not what is expected: \n%v\n%v", null, []byte{128, 0, 0, 0, 0, 0, 0, 0})
	} else if !reflect.DeepEqual(pos, []byte{128, 0, 0, 0, 0, 0, 0, 1}) {
		t.Errorf("positive values is not what is expected: \n%v\n%v", pos, []byte{128, 0, 0, 0, 0, 0, 0, 1})
	}

	if bytes.Compare(neg, null) >= 0 {
		t.Error("negative values are not smaller than null", neg, null)
	} else if bytes.Compare(null, pos) >= 0 {
		t.Error("null values are not smaller than positive", null, pos)
	} else if bytes.Compare(neg, pos) >= 0 {
		t.Error("negative values are not smaller than positive", neg, pos)
	}

	neg, _ = intToBytes(int64(math.MinInt64))
	null, _ = intToBytes(int64(0))
	pos, _ = intToBytes(int64(math.MaxInt64))

	if !reflect.DeepEqual(neg, []byte{0, 0, 0, 0, 0, 0, 0, 0}) {
		t.Errorf("negative values is not what is expected: \n%v\n%v", neg, []byte{0, 0, 0, 0, 0, 0, 0, 0})
	} else if !reflect.DeepEqual(null, []byte{128, 0, 0, 0, 0, 0, 0, 0}) {
		t.Errorf("null values is not what is expected: \n%v\n%v", null, []byte{128, 0, 0, 0, 0, 0, 0, 0})
	} else if !reflect.DeepEqual(pos, []byte{255, 255, 255, 255, 255, 255, 255, 255}) {
		t.Errorf("positive values is not what is expected: \n%v\n%v", pos, []byte{255, 255, 255, 255, 255, 255, 255, 255})
	}

	if bytes.Compare(neg, null) >= 0 {
		t.Error("negative values are not smaller than null", neg, null)
	} else if bytes.Compare(null, pos) >= 0 {
		t.Error("null values are not smaller than positive", null, pos)
	} else if bytes.Compare(neg, pos) >= 0 {
		t.Error("negative values are not smaller than positive", neg, pos)
	}

	if _, err := intToBytes(time.Now()); err == nil {
		t.Error(err)
		return
	}
}

func TestFloatConversion(t *testing.T) {
	v := 31497.545785
	if _, err := floatToBytes(v); err != nil {
		t.Error(err)
		return
	}

	v = -3223597.00004855
	if _, err := floatToBytes(v); err != nil {
		t.Error(err)
		return
	}

	v = 0.9
	if _, err := floatToBytes(v); err != nil {
		t.Error(err)
		return
	}

	v = -0.9
	if _, err := floatToBytes(v); err != nil {
		t.Error(err)
		return
	}
}

// func TestFloatOrdering(t *testing.T) {
// 	neg, _ := floatToBytes(-1.0)
// 	null, _ := floatToBytes(0.0)
// 	pos, _ := floatToBytes(1.0)

// 	if !reflect.DeepEqual(neg, []byte{63, 240, 0, 0, 0, 0, 0, 0}) {
// 		t.Errorf("negative values is not what is expected: \n%v\n%v", neg, []byte{63, 240, 0, 0, 0, 0, 0, 0})
// 	} else if !reflect.DeepEqual(null, []byte{128, 0, 0, 0, 0, 0, 0, 0}) {
// 		t.Errorf("null values is not what is expected: \n%v\n%v", null, []byte{128, 0, 0, 0, 0, 0, 0, 0})
// 	} else if !reflect.DeepEqual(pos, []byte{191, 240, 0, 0, 0, 0, 0, 0}) {
// 		t.Errorf("positive values is not what is expected: \n%v\n%v", pos, []byte{191, 240, 0, 0, 0, 0, 0, 0})
// 	}

// 	if bytes.Compare(neg, null) >= 0 {
// 		t.Error("negative values are not smaller than null", neg, null)
// 	} else if bytes.Compare(null, pos) >= 0 {
// 		t.Error("null values are not smaller than positive", null, pos)
// 	} else if bytes.Compare(neg, pos) >= 0 {
// 		t.Error("negative values are not smaller than positive", neg, pos)
// 	}

// 	neg, _ = floatToBytes(math.SmallestNonzeroFloat64)
// 	null, _ = floatToBytes(0.0)
// 	pos, _ = floatToBytes(math.MaxFloat64)

// 	if !reflect.DeepEqual(neg, []byte{128, 0, 0, 0, 0, 0, 0, 1}) {
// 		t.Errorf("negative values is not what is expected: \n%v\n%v", neg, []byte{128, 0, 0, 0, 0, 0, 0, 1})
// 	} else if !reflect.DeepEqual(null, []byte{128, 0, 0, 0, 0, 0, 0, 0}) {
// 		t.Errorf("null values is not what is expected: \n%v\n%v", null, []byte{128, 0, 0, 0, 0, 0, 0, 0})
// 	} else if !reflect.DeepEqual(pos, []byte{255, 239, 255, 255, 255, 255, 255, 255}) {
// 		t.Errorf("positive values is not what is expected: \n%v\n%v", pos, []byte{255, 239, 255, 255, 255, 255, 255, 255})
// 	}

// 	if bytes.Compare(neg, null) >= 0 {
// 		t.Error("negative values are not smaller than null", neg, null)
// 	} else if bytes.Compare(null, pos) >= 0 {
// 		t.Error("null values are not smaller than positive", null, pos)
// 	} else if bytes.Compare(neg, pos) >= 0 {
// 		t.Error("negative values are not smaller than positive", neg, pos)
// 	}

// 	if _, err := intToBytes(time.Now()); err == nil {
// 		t.Error(err)
// 		return
// 	}
// }

func TestTimeConversion(t *testing.T) {
	if _, err := timeToBytes(time.Now()); err != nil {
		t.Error(err)
		return
	}

	if _, err := timeToBytes("is it time?"); err == nil {
		t.Error(err)
		return
	}
}
