package footer

import (
	"bytes"
	"time"

	"github.com/radu-matei/azure-vhd-utils/vhdcore"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/common"
)

// Footer represents the footer of the vhd, the size of the footer is 512 bytes.
// The last 512 bytes of the disk is footer.  In case of dynamic and differential
// vhds, the footer is replicated at the beginning of the disk as well.
//
type Footer struct {
	// Offset =  0, Size = 8
	Cookie *vhdcore.Cookie
	// Offset =  8, Size = 4
	Features VhdFeature
	// Offset = 12, Size = 4
	FileFormatVersion VhdFileFormatVersion
	// Offset = 16, Size = 8
	// Absolute byte offset to the header structure, this is  used for dynamic disks
	// and differencing disks. Fixed disk does not have header this field is set to
	// 0xFFFFFFFF for fixed disk.
	HeaderOffset int64
	// Offset = 24, Size = 4
	TimeStamp *time.Time
	// Offset = 28, Size = 4
	CreatorApplication string
	// Offset = 32, Size = 4
	CreatorVersion VhdCreatorVersion
	// Offset = 36, Size = 4
	CreatorHostOsType HostOsType
	// Offset = 40, Size = 8
	PhysicalSize int64
	// Offset = 48, Size = 8
	VirtualSize int64
	// Offset = 56, Size = 4
	DiskGeometry *DiskGeometry
	// Offset = 60, Size = 4
	DiskType DiskType
	// Offset = 64, Size = 4
	CheckSum uint32
	// Offset = 68, Size = 16
	UniqueID *common.UUID
	// Offset = 84, Size = 1
	SavedState bool
	// Offset = 85, Size = 427
	Reserved []byte
	// Offset = 0, Size = 512
	RawData []byte
}

// CreateCopy creates and returns a deep copy of this instance.
//
func (v *Footer) CreateCopy() *Footer {
	return &Footer{
		Cookie:             v.Cookie.CreateCopy(),
		Features:           v.Features,
		FileFormatVersion:  v.FileFormatVersion,
		HeaderOffset:       v.HeaderOffset,
		TimeStamp:          v.TimeStamp,
		CreatorApplication: v.CreatorApplication,
		CreatorVersion:     v.CreatorVersion,
		CreatorHostOsType:  v.CreatorHostOsType,
		PhysicalSize:       v.PhysicalSize,
		VirtualSize:        v.VirtualSize,
		DiskGeometry:       v.DiskGeometry.CreateCopy(),
		DiskType:           v.DiskType,
		CheckSum:           v.CheckSum,
		UniqueID:           v.UniqueID,
		SavedState:         v.SavedState,
		Reserved:           common.CreateByteSliceCopy(v.Reserved),
		RawData:            common.CreateByteSliceCopy(v.RawData),
	}
}

// Equal returns true if this and other points to the same instance or if contents
// of the fields of these two instances are same.
//
func (v *Footer) Equal(other *Footer) bool {
	if other == nil {
		return false
	}

	if v == other {
		return true
	}

	return v.Cookie.Equal(other.Cookie) &&
		v.Features == other.Features &&
		v.FileFormatVersion == other.FileFormatVersion &&
		v.HeaderOffset == other.HeaderOffset &&
		v.TimeStamp == other.TimeStamp &&
		v.CreatorApplication == other.CreatorApplication &&
		v.CreatorVersion == other.CreatorVersion &&
		v.CreatorHostOsType == other.CreatorHostOsType &&
		v.PhysicalSize == other.PhysicalSize &&
		v.VirtualSize == other.VirtualSize &&
		v.DiskGeometry.Equals(other.DiskGeometry) &&
		v.DiskType == other.DiskType &&
		v.CheckSum == other.CheckSum &&
		v.UniqueID == other.UniqueID &&
		v.SavedState == other.SavedState &&
		bytes.Equal(v.Reserved, other.Reserved) &&
		bytes.Equal(v.RawData, other.RawData)
}
