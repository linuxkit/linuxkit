package footer

import "fmt"

// VhdCreatorVersion represents the major/minor version of the application that
// created the hard disk image. The version is stored in the vhd footer in
// big-endian format.
//
type VhdCreatorVersion uint32

const (
	// VhdCreatorVersionNone represents a nil host Creator version
	VhdCreatorVersionNone VhdCreatorVersion = 0
	// VhdCreatorVersionVS2004 represents the value set by Virtual Server 2004
	VhdCreatorVersionVS2004 VhdCreatorVersion = 0x00010000
	// VhdCreatorVersionVPC2004 represents the value set by Virtual PC 2004
	VhdCreatorVersionVPC2004 VhdCreatorVersion = 0x00050000
	// VhdCreatorVersionCSUP2011 represents a value set by CSUP 2011
	VhdCreatorVersionCSUP2011 VhdCreatorVersion = 0x00070000
)

// String returns the string representation of the VhdCreatorVersion. If the int
// VhdCreatorVersion value does not match with the predefined CreatorVersions then
// this function convert the int to string and return.
//
func (v VhdCreatorVersion) String() string {
	switch v {
	case VhdCreatorVersionVS2004:
		return "VS2004"
	case VhdCreatorVersionVPC2004:
		return "VPC2004"
	case VhdCreatorVersionCSUP2011:
		return "SUP2011"
	}

	return fmt.Sprintf("%d", v)
}
