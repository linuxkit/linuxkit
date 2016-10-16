package instance

import "encoding/json"

// Plugin is a vendor-agnostic API used to create and manage resources with an infrastructure provider.
type Plugin interface {
	// Validate performs local validation on a provision request.
	Validate(req json.RawMessage) error

	// Provision creates a new instance based on the spec.
	Provision(spec Spec) (*ID, error)

	// Destroy terminates an existing instance.
	Destroy(instance ID) error

	// DescribeInstances returns descriptions of all instances matching all of the provided tags.
	DescribeInstances(tags map[string]string) ([]Description, error)
}
