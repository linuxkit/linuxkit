package metadata

import (
	"github.com/docker/infrakit/pkg/spi"
	"github.com/docker/infrakit/pkg/types"
)

// InterfaceSpec is the current name and version of the Metadata API.
var InterfaceSpec = spi.InterfaceSpec{
	Name:    "Metadata",
	Version: "0.1.0",
}

// Plugin is the interface for metadata-related operations.
type Plugin interface {

	// List returns a list of *child nodes* given a path, which is specified as a slice
	List(path Path) (child []string, err error)

	// Get retrieves the value at path given.
	Get(path Path) (value *types.Any, err error)
}
