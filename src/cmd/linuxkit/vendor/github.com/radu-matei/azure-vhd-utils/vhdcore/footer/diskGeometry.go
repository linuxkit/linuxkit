package footer

import (
	"fmt"

	"github.com/radu-matei/azure-vhd-utils/vhdcore"
)

// DiskGeometry represents the cylinder, heads and sectors (CHS) per track.
//
type DiskGeometry struct {
	// Offset = 0, Size = 2
	// Stored in big-endian format
	Cylinder uint16
	// Offset = 2, Size = 1
	Heads byte
	// Offset = 3, Size = 1
	Sectors byte
}

// CreateNewDiskGeometry creates a new DiskGeometry from the given virtual
// size. CHS field values are calculated based on the total data sectors
// present in the disk image.
//
func CreateNewDiskGeometry(virtualSize int64) *DiskGeometry {
	// Total data sectors present in the disk image
	var totalSectors = virtualSize / vhdcore.VhdSectorLength
	// Sectors per track on the disk
	var sectorsPerTrack int64
	// Number of heads present on the disk
	var heads int32
	// Cylinders * heads
	var cylinderTimesHeads int64

	//                  C   * H  * S
	if totalSectors > 65535*16*255 {
		totalSectors = 65535 * 16 * 255
	}

	if totalSectors >= 65535*16*63 {
		sectorsPerTrack = 255
		cylinderTimesHeads = totalSectors / sectorsPerTrack
		heads = 16

		return &DiskGeometry{
			Cylinder: uint16(cylinderTimesHeads / int64(heads)),
			Heads:    byte(heads),
			Sectors:  byte(sectorsPerTrack),
		}
	}

	sectorsPerTrack = 17
	cylinderTimesHeads = totalSectors / sectorsPerTrack
	heads = int32((cylinderTimesHeads + 1023) / 1024)

	if heads < 4 {
		heads = 4
	}

	if cylinderTimesHeads >= int64(heads*1024) || heads > 16 {
		sectorsPerTrack = 31
		heads = 16
		cylinderTimesHeads = totalSectors / sectorsPerTrack
	}

	if cylinderTimesHeads >= int64(heads*1024) {
		sectorsPerTrack = 63
		heads = 16
		cylinderTimesHeads = totalSectors / sectorsPerTrack
	}

	return &DiskGeometry{
		Cylinder: uint16(cylinderTimesHeads / int64(heads)),
		Heads:    byte(heads),
		Sectors:  byte(sectorsPerTrack),
	}
}

// CreateCopy creates a copy of this instance
//
func (d *DiskGeometry) CreateCopy() *DiskGeometry {
	return &DiskGeometry{
		Cylinder: d.Cylinder,
		Heads:    d.Heads,
		Sectors:  d.Sectors,
	}
}

// Equals returns true if this and other points to the same instance
// or if CHS fields of pointed instances are same
//
func (d *DiskGeometry) Equals(other *DiskGeometry) bool {
	if other == nil {
		return false
	}

	return other == d || *other == *d
}

// String returns the string representation of this range, this satisfies stringer interface.
//
func (d *DiskGeometry) String() string {
	return fmt.Sprintf("Cylinder:%d Heads:%d Sectors:%d", d.Cylinder, d.Heads, d.Sectors)
}
