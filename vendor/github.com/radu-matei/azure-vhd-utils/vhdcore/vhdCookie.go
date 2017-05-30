package vhdcore

import "bytes"

// VhdFooterCookie is the cookie value stored in VHD footer
// Microsoft uses the “conectix” string to identify a hard disk image created by
// Microsoft Virtual Server, Virtual PC, and predecessor products. The cookie is
// stored as an eight-character ASCII string with the “c” in the first byte,
// the “o” in the second byte, and so on.
//
const VhdFooterCookie = "conectix"

// VhdHeaderCookie is the header cookie which is always cxsparse
//
const VhdHeaderCookie = "cxsparse"

// Cookie represents the Vhd header or Vhd footer cookie.
// Footer Cookie are used to uniquely identify the original creator of the hard disk
// image. The values are case-sensitive. Header Cookie holds the value "cxsparse".
type Cookie struct {
	Data     []byte
	isHeader bool
}

// CreateNewVhdCookie creates a new VhdCookie, the new instance's Data will be simply
// reference to the byte slice data (i.e. this function will not create a copy)
//
func CreateNewVhdCookie(isHeader bool, data []byte) *Cookie {
	return &Cookie{isHeader: isHeader, Data: data}
}

// CreateFooterCookie creates a VhdCookie representing vhd footer cookie
//
func CreateFooterCookie() *Cookie {
	return CreateNewVhdCookie(false, []byte(VhdFooterCookie))
}

// CreateHeaderCookie creates a VhdCookie representing vhd header cookie
//
func CreateHeaderCookie() *Cookie {
	return CreateNewVhdCookie(true, []byte(VhdHeaderCookie))
}

// IsValid checks whether this this instance's internal cookie string is valid.
//
func (c *Cookie) IsValid() bool {
	if c.isHeader {
		return bytes.Equal(c.Data, []byte(VhdHeaderCookie))
	}

	return bytes.Equal(c.Data, []byte(VhdFooterCookie))
}

// CreateCopy creates a copy of this instance
//
func (c *Cookie) CreateCopy() *Cookie {
	cp := &Cookie{isHeader: c.isHeader}
	cp.Data = make([]byte, len(c.Data))
	copy(cp.Data, c.Data)
	return cp
}

// Equal returns true if this and other points to the same instance or contents of field
// values of two are same.
//
func (c *Cookie) Equal(other *Cookie) bool {
	if other == nil {
		return false
	}

	if c == other {
		return true
	}

	return c.isHeader == other.isHeader && bytes.Equal(c.Data, other.Data)
}

// String returns the string representation of this range, this satisfies stringer interface.
//
func (c *Cookie) String() string {
	return string(c.Data)
}
