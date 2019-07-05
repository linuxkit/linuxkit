package utils

import "time"

// String returns a pointer to the string value passed in.
func String(v string) *string {
	return &v
}

// StringSlice converts a slice of string values into a slice of
// string pointers
func StringSlice(src []string) []*string {
	dst := make([]*string, len(src))
	for i := 0; i < len(src); i++ {
		dst[i] = &(src[i])
	}
	return dst
}

// Strings returns a pointer to the []string value passed in.
func Strings(v []string) *[]string {
	return &v
}

// StringsSlice converts a slice of []string values into a slice of
// []string pointers
func StringsSlice(src [][]string) []*[]string {
	dst := make([]*[]string, len(src))
	for i := 0; i < len(src); i++ {
		dst[i] = &(src[i])
	}
	return dst
}

// Bytes returns a pointer to the []byte value passed in.
func Bytes(v []byte) *[]byte {
	return &v
}

// BytesSlice converts a slice of []byte values into a slice of
// []byte pointers
func BytesSlice(src [][]byte) []*[]byte {
	dst := make([]*[]byte, len(src))
	for i := 0; i < len(src); i++ {
		dst[i] = &(src[i])
	}
	return dst
}

// Bool returns a pointer to the bool value passed in.
func Bool(v bool) *bool {
	return &v
}

// BoolSlice converts a slice of bool values into a slice of
// bool pointers
func BoolSlice(src []bool) []*bool {
	dst := make([]*bool, len(src))
	for i := 0; i < len(src); i++ {
		dst[i] = &(src[i])
	}
	return dst
}

// Int32 returns a pointer to the int32 value passed in.
func Int32(v int32) *int32 {
	return &v
}

// Int32Slice converts a slice of int32 values into a slice of
// int32 pointers
func Int32Slice(src []int32) []*int32 {
	dst := make([]*int32, len(src))
	for i := 0; i < len(src); i++ {
		dst[i] = &(src[i])
	}
	return dst
}

// Int64 returns a pointer to the int64 value passed in.
func Int64(v int64) *int64 {
	return &v
}

// Int64Slice converts a slice of int64 values into a slice of
// int64 pointers
func Int64Slice(src []int64) []*int64 {
	dst := make([]*int64, len(src))
	for i := 0; i < len(src); i++ {
		dst[i] = &(src[i])
	}
	return dst
}

// Uint32 returns a pointer to the uint32 value passed in.
func Uint32(v uint32) *uint32 {
	return &v
}

// Uint32Slice converts a slice of uint32 values into a slice of
// uint32 pointers
func Uint32Slice(src []uint32) []*uint32 {
	dst := make([]*uint32, len(src))
	for i := 0; i < len(src); i++ {
		dst[i] = &(src[i])
	}
	return dst
}

// Uint64 returns a pointer to the uint64 value passed in.
func Uint64(v uint64) *uint64 {
	return &v
}

// Uint64Slice converts a slice of uint64 values into a slice of
// uint64 pointers
func Uint64Slice(src []uint64) []*uint64 {
	dst := make([]*uint64, len(src))
	for i := 0; i < len(src); i++ {
		dst[i] = &(src[i])
	}
	return dst
}

// Float32 returns a pointer to the float32 value passed in.
func Float32(v float32) *float32 {
	return &v
}

// Float32Slice converts a slice of float32 values into a slice of
// float32 pointers
func Float32Slice(src []float32) []*float32 {
	dst := make([]*float32, len(src))
	for i := 0; i < len(src); i++ {
		dst[i] = &(src[i])
	}
	return dst
}

// Float64 returns a pointer to the float64 value passed in.
func Float64(v float64) *float64 {
	return &v
}

// Float64Slice converts a slice of float64 values into a slice of
// float64 pointers
func Float64Slice(src []float64) []*float64 {
	dst := make([]*float64, len(src))
	for i := 0; i < len(src); i++ {
		dst[i] = &(src[i])
	}
	return dst
}

// Duration returns a pointer to the Duration value passed in.
func Duration(v time.Duration) *time.Duration {
	return &v
}
