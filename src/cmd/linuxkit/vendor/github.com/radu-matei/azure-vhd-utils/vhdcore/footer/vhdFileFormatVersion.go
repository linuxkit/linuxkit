package footer

// VhdFileFormatVersion represents the version of the specification used in creating
// the vhd. The version is stored in the vhd footer in big-endian format.
// This is a 4 byte value - most-significant two bytes are for the major version.
// The least-significant two bytes are the minor version
//
type VhdFileFormatVersion uint32

// VhdFileFormatVersionDefault represents the currently supported vhd specification version.
//
const VhdFileFormatVersionDefault VhdFileFormatVersion = 0x00010000

// VhdFileFormatVersionNone represents invalid version
//
const VhdFileFormatVersionNone VhdFileFormatVersion = 0

// IsSupported returns true if this instance represents a supported vhd specification
// version.
//
func (v VhdFileFormatVersion) IsSupported() bool {
	return v == VhdFileFormatVersionDefault
}
