package parentlocator

import (
	"github.com/radu-matei/azure-vhd-utils/vhdcore/common"
	"log"
	"strings"
)

// ParentLocator represents an entry in Parent locator table. Each entry represents
// details (parent-hard-disk-locator-info) of file locator which is used to locate
// the parent disk file of differencing hard disk.
//
type ParentLocator struct {
	// Offset = 0, Size = 4
	// This field stores the code representing the platform-specific format used for
	// the file locator. Stored in big-endian format.
	PlatformCode PlatformCode

	// Offset = 4, Size = 4
	// This field stores the number of 512-byte sectors needed to store the parent
	// hard disk locator. Stored in big-endian format.
	PlatformDataSpace int32

	// Offset = 8, Size = 4
	// This field stores the actual length of the parent hard disk locator in bytes.
	// Stored in big-endian format.
	PlatformDataLength int32

	// Offset = 12, Size = 4
	// This field must be set to zero.
	// Stored in big-endian format.
	Reserved int32

	// Offset = 16, Size = 8
	// This field stores the absolute file offset in bytes where the platform specific
	// file locator data is stored. Stored in big-endian format.
	PlatformDataOffset int64

	// This is not a field that get stored or retrieved from disk's ParentLocator entry.
	// We use this field to store the resolved file locator path.
	PlatformSpecificFileLocator string
}

// SetPlatformSpecificFileLocator retrieves the file locator value and store that in the property
// PlatformSpecificFileLocator
//
func (l *ParentLocator) SetPlatformSpecificFileLocator(fileLocator []byte) {
	// 1. For the platform codes - W2Ru and W2Ku, fileLocator contents is UTF-16 encoded.
	// 2. For the platform code  - MacX,          fileLocator contents is UTF-8 encoded.
	// 3. For unknown platform code               fileLocator contents is treated as UTF-16 encoded.
	//
	// For 1 the byte order is little-endian
	// For 3 the byte order is big-endian

	if l.PlatformCode == PlatformCodeWi2R || l.PlatformCode == PlatformCodeWi2K {
		log.Panicf("Deprecated PlatformCode: %d", l.PlatformCode)
	}

	if l.PlatformCode == PlatformCodeMac {
		log.Panicf("Handling Mac OS alias stored as a blob is not implemented, PlatformCode: %d", l.PlatformCode)
	}

	if l.PlatformCode == PlatformCodeNone {
		l.PlatformSpecificFileLocator = ""
	} else if l.PlatformCode == PlatformCodeW2Ru {
		//TODO: Add differencing disks path name, this is relative path
		l.PlatformSpecificFileLocator = common.Utf16BytesToStringLE(fileLocator)
	} else if l.PlatformCode == PlatformCodeW2Ku {
		l.PlatformSpecificFileLocator = common.Utf16BytesToStringLE(fileLocator)
	} else if l.PlatformCode == PlatformCodeMacX {
		l.PlatformSpecificFileLocator = string(fileLocator)
	} else {
		l.PlatformSpecificFileLocator = strings.TrimSuffix(common.Utf16BytesToStringBE(fileLocator), "\x00")
	}
}
