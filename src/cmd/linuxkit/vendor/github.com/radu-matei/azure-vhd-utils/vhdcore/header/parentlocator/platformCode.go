package parentlocator

import "fmt"

// The PlatformCode describes which platform-specific format is used for the file locator
// This is the type of PlatformCode field in ParentLocator type.
//
type PlatformCode int32

const (
	// PlatformCodeNone indicates a nil value for platform code.
	//
	PlatformCodeNone PlatformCode = 0x0
	// PlatformCodeWi2R [deprecated]
	//
	PlatformCodeWi2R = 0x57693272
	// PlatformCodeWi2K [deprecated]
	//
	PlatformCodeWi2K = 0x5769326B
	// PlatformCodeW2Ru indicate that the file locator is stored in unicode (UTF-16) format on Windows
	// relative to the differencing disk pathname.
	//
	PlatformCodeW2Ru = 0x57327275
	// PlatformCodeW2Ku indicate that the file locator is stored in unicode (UTF-16) format as absolute
	// pathname on Windows.
	//
	PlatformCodeW2Ku = 0x57326B75
	// PlatformCodeMac indicates that file locator is a Mac OS alias stored as a blob.
	//
	PlatformCodeMac = 0x4D616320
	// PlatformCodeMacX indicates that file locator is a file URL with UTF-8 encoding conforming to RFC 2396.
	//
	PlatformCodeMacX = 0x4D616358
)

// String returns the string representation of the PlatformCode. If the int platform code
// value does not match with the predefined PlatformCodes then this function convert the
// int to string and return
//
func (p PlatformCode) String() string {
	switch p {
	case PlatformCodeNone:
		return "None"
	case PlatformCodeWi2R:
		return "Wi2R [deprecated]"
	case PlatformCodeWi2K:
		return "Wi2K [deprecated]"
	case PlatformCodeW2Ru:
		return "W2Ru"
	case PlatformCodeW2Ku:
		return "W2Ku"
	case PlatformCodeMac:
		return "Mac"
	case PlatformCodeMacX:
		return "MacX"
	}

	return fmt.Sprint("%d", p)
}
