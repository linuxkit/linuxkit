package common

import (
	"encoding/binary"
	"unicode/utf16"
	"unicode/utf8"
)

// Utf16BytesToStringLE decode the given UTF16 encoded byte sequence and returns
// Go UTF8 encoded string, the byte order of the given sequence is little-endian.
//
func Utf16BytesToStringLE(b []byte) string {
	return Utf16BytesToString(b, binary.LittleEndian)
}

// Utf16BytesToStringBE decode the given UTF16 encoded byte sequence and returns
// Go UTF8 encoded string, the byte order of the given sequence is big-endian.
//
func Utf16BytesToStringBE(b []byte) string {
	return Utf16BytesToString(b, binary.BigEndian)
}

// Utf16BytesToString decode the given UTF16 encoded byte sequence and returns
// Go UTF8 encoded string, the byte order of the sequence is determined by the
// given binary.ByteOrder parameter.
//
func Utf16BytesToString(b []byte, o binary.ByteOrder) string {
	var u []uint16
	l := len(b)
	if l&1 == 0 {
		u = make([]uint16, l>>1)
	} else {
		u = make([]uint16, l>>1+1)
		u[len(u)-1] = utf8.RuneError
	}

	for i, j := 0, 0; j+1 < l; i, j = i+1, j+2 {
		u[i] = o.Uint16(b[j:])
	}

	return string(utf16.Decode(u))
}

// CreateByteSliceCopy creates and returns a copy of the given slice.
//
func CreateByteSliceCopy(b []byte) []byte {
	r := make([]byte, len(b))
	copy(r, b)
	return r
}
