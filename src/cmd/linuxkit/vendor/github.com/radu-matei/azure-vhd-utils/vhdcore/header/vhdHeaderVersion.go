package header

// VhdHeaderVersion represents the  major/minor version of the specification
// used in creating the vhd. The version is stored in the vhd header in
// big-endian format.
//
type VhdHeaderVersion uint32

// VhdHeaderSupportedVersion indicates the current VHD specification version
//
const VhdHeaderSupportedVersion VhdHeaderVersion = 0x00010000

// VhdHeaderVersionNone indicates an invalid VHD specification version
//
const VhdHeaderVersionNone = 0

// IsSupported returns true if this instance represents a supported VHD specification
// version.
//
func (v VhdHeaderVersion) IsSupported() bool {
	return v == VhdHeaderSupportedVersion
}
