package footer

// DiskType type represents the type of the disk, Value is stored in the footer
// in big-endian format.
//
type DiskType uint32

const (
	// DiskTypeNone represents a nil disk type
	//
	DiskTypeNone DiskType = 0
	// DiskTypeFixed represents a fixed disk type
	//
	DiskTypeFixed DiskType = 2
	// DiskTypeDynamic represents a dynamic disk type
	//
	DiskTypeDynamic DiskType = 3
	// DiskTypeDifferencing represents a differencing disk type
	//
	DiskTypeDifferencing DiskType = 4
)

// String returns the string representation of the DiskType. If the int type value
// does not match with the predefined disk types then this function return the
// string "UnknownDiskType"
//
func (d DiskType) String() string {
	switch d {
	case DiskTypeFixed:
		return "Fixed"
	case DiskTypeDynamic:
		return "Dynamic"
	case DiskTypeDifferencing:
		return "Differencing"
	}

	return "UnknownDiskType"
}
