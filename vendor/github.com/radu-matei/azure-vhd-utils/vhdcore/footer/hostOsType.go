package footer

import "fmt"

// HostOsType represents the host operating system a disk image is created on.
// Value is stored in the footer in big-endian format.
//
type HostOsType uint32

const (
	// HostOsTypeNone represents a nil host OS type
	HostOsTypeNone HostOsType = 0
	// HostOsTypeWindows represents a Windows OS type
	HostOsTypeWindows HostOsType = 0x5769326B
	// HostOsTypeMacintosh represents a MAC OS type
	HostOsTypeMacintosh HostOsType = 0x4D616320
)

// String returns the string representation of the HostOsType. If the int type
// value does not match with the predefined OS types then this function convert
// the int to string and return
//
func (h HostOsType) String() string {
	switch h {
	case HostOsTypeWindows:
		return "Windows"
	case HostOsTypeMacintosh:
		return "Macintosh"
	}

	return fmt.Sprintf("%d", h)
}
