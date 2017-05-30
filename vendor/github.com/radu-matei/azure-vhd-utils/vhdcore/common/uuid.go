package common

import (
	"errors"
	"fmt"
)

// UUID represents a Universally Unique Identifier.
//
type UUID struct {
	uuid [16]byte
}

// NewUUID creates a new UUID, it uses the given 128-bit (16 byte) value as the uuid.
//
func NewUUID(b []byte) (*UUID, error) {
	if len(b) != 16 {
		return nil, errors.New("NewUUID: buffer requires to be 16 bytes")
	}

	u := &UUID{}
	copy(u.uuid[:], b)
	return u, nil
}

// String returns the string representation of the UUID which is 16 hex digits separated by hyphens
// int form xxxx-xx-xx-xx-xxxxxx
//
func (u *UUID) String() string {
	a := uint32(u.uuid[3])<<24 | uint32(u.uuid[2])<<16 | uint32(u.uuid[1])<<8 | uint32(u.uuid[0])
	// a := b.order.Uint32(b.buffer[:4])
	b1 := int16(int32(u.uuid[5])<<8 | int32(u.uuid[4]))
	// b1 := b.order.Uint16(b.buffer[4:6])
	c := int16(int32(u.uuid[7])<<8 | int32(u.uuid[6]))
	// c := b.order.Uint16(b.buffer[6:8])
	d := u.uuid[8]
	e := u.uuid[9]
	f := u.uuid[10]
	g := u.uuid[11]
	h := u.uuid[12]
	i := u.uuid[13]
	j := u.uuid[14]
	k := u.uuid[15]
	return fmt.Sprintf("%x-%x-%x-%x%x-%x%x%x%x%x%x", a, b1, c, d, e, f, g, h, i, j, k)
}

// ToByteSlice returns the UUID as byte slice.
//
func (u *UUID) ToByteSlice() []byte {
	b := make([]byte, 16)
	copy(b, u.uuid[:])
	return b
}
