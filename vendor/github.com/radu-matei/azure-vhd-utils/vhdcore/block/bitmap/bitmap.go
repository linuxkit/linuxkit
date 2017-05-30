package bitmap

import "fmt"

// BitMap type represents 'bitmap section' of a block.
//
type BitMap struct {
	data   []byte
	Length int32
}

// NewBitMapFromByteSlice creates a new BitMap, b is the byte slice that needs to be used as bitmap
// source. The caller should not reuse this byte slice anymore.
//
func NewBitMapFromByteSlice(b []byte) *BitMap {
	return &BitMap{data: b, Length: int32(len(b)) * 8}
}

// NewBitMapFromByteSliceCopy creates a new BitMap, b is the byte slice that needs to be used as bitmap
// source. The caller can reuse the byte slice as this method creates a copy of it.
//
func NewBitMapFromByteSliceCopy(b []byte) *BitMap {
	data := make([]byte, len(b))
	copy(data, b)
	return &BitMap{data: data, Length: int32(len(b)) * 8}
}

// Set sets the bit at the given index. It returns error if idx < 0 or idx >= bitsCount.
//
func (b *BitMap) Set(idx int32, value bool) error {
	if idx < 0 || idx >= b.Length {
		return fmt.Errorf("The index %d is out of boundary", idx)
	}

	i := idx >> 3
	m := 1 << (uint32(idx) & 7)

	if value {
		b.data[i] = b.data[i] | byte(m)
	} else {
		b.data[i] = b.data[i] & byte(^m)
	}

	return nil
}

// Get returns the value of the bit at the given index. It returns error if idx < 0 or idx >= bitsCount.
//
func (b *BitMap) Get(idx int32) (bool, error) {
	if idx < 0 || idx >= b.Length {
		return false, fmt.Errorf("The index %d is out of boundary", idx)
	}

	i := idx >> 3
	m := 1 << (uint32(idx) & 7)

	return (b.data[i] & byte(m)) != 0, nil
}
