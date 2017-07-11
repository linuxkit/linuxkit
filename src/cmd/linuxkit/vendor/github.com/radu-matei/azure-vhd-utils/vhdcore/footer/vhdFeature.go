package footer

import "fmt"

// VhdFeature represents a bit field used to indicate specific feature support.
// Value is stored in the footer in big-endian format.
//
type VhdFeature uint32

const (
	// VhdFeatureNoFeaturesEnabled indicates that hard disk image has no special features enabled in it.
	//
	VhdFeatureNoFeaturesEnabled VhdFeature = 0x00000000
	// VhdFeatureTemporary indicates that current disk is a temporary disk. A temporary disk designation
	// indicates to an application that this disk is a candidate for deletion on shutdown.
	//
	VhdFeatureTemporary = 0x00000001
	// VhdFeatureReserved represents a bit must always be set to 1. All other bits are also reserved
	// and should be set to 0
	//
	VhdFeatureReserved = 0x00000002
)

// String returns the string representation of the VhdFeature. If the int VhdFeature
// value does not match with the predefined VhdFeatures then this function convert
// int to string and return
//
func (v VhdFeature) String() string {
	switch v {
	case VhdFeatureNoFeaturesEnabled:
		return "NoFeaturesEnabled"
	case VhdFeatureTemporary:
		return "Temporary"
	case VhdFeatureReserved:
		return "Reserved"
	}

	return fmt.Sprint("%d", v)
}
