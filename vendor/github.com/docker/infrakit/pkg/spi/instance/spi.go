package instance

import (
	"github.com/docker/infrakit/pkg/spi"
	"github.com/docker/infrakit/pkg/types"
)

// InterfaceSpec is the current name and version of the Instance API.
var InterfaceSpec = spi.InterfaceSpec{
	Name:    "Instance",
	Version: "0.3.0",
}

// Plugin is a vendor-agnostic API used to create and manage resources with an infrastructure provider.
type Plugin interface {
	// Validate performs local validation on a provision request.
	Validate(req *types.Any) error

	// Provision creates a new instance based on the spec.
	Provision(spec Spec) (*ID, error)

	// Label labels the instance
	Label(instance ID, labels map[string]string) error

	// Destroy terminates an existing instance.
	Destroy(instance ID) error

	// DescribeInstances returns descriptions of all instances matching all of the provided tags.
	DescribeInstances(labels map[string]string) ([]Description, error)
}
