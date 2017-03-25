package spi

import (
	"github.com/docker/infrakit/pkg/types"
)

// InterfaceSpec is metadata about an API.
type InterfaceSpec struct {
	// Name of the interface.
	Name string

	// Version is the identifier for the API version.
	Version string
}

// VendorInfo provides vendor-specific information
type VendorInfo struct {
	InterfaceSpec // vendor-defined name / version

	// URL is the informational url for the plugin. It can container help and docs, etc.
	URL string
}

// Vendor is an optional interface that has vendor-specific information methods
type Vendor interface {
	// VendorInfo returns a vendor-defined interface spec
	VendorInfo() *VendorInfo
}

// InputExample interface is an optional interface implemented by the plugin that will provide
// example input struct to document the vendor-specific api of the plugin. An example of this
// is to provide a sample JSON for all the Properties field in the plugin API.
type InputExample interface {

	// ExampleProperties returns an example JSON raw message that the vendor plugin understands.
	// This is an example of what the user will configure and what will be used as the opaque
	// blob in all the plugin methods where raw JSON messages are referenced.
	ExampleProperties() *types.Any
}
