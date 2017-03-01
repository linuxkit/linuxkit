package vanilla

import (
	"encoding/json"
	"strings"

	"github.com/docker/infrakit/plugin/group/types"
	"github.com/docker/infrakit/spi/flavor"
	"github.com/docker/infrakit/spi/instance"
)

// Spec is the model of the Properties section of the top level group spec.
type Spec struct {
	// Init
	Init []string

	// Tags
	Tags map[string]string
}

// NewPlugin creates a Flavor plugin that doesn't do very much. It assumes instances are
// identical (cattles) but can assume specific identities (via the LogicalIDs).  The
// instances here are treated identically because we have constant Init that applies
// to all instances
func NewPlugin() flavor.Plugin {
	return vanillaFlavor(0)
}

type vanillaFlavor int

func (f vanillaFlavor) Validate(flavorProperties json.RawMessage, allocation types.AllocationMethod) error {
	return json.Unmarshal(flavorProperties, &Spec{})
}

func (f vanillaFlavor) Healthy(flavorProperties json.RawMessage, inst instance.Description) (flavor.Health, error) {
	// TODO: We could add support for shell code in the Spec for a command to run for checking health.
	return flavor.Healthy, nil
}

func (f vanillaFlavor) Prepare(
	flavor json.RawMessage,
	instance instance.Spec,
	allocation types.AllocationMethod) (instance.Spec, error) {

	s := Spec{}
	err := json.Unmarshal(flavor, &s)
	if err != nil {
		return instance, err
	}

	// Append Init
	lines := []string{}
	if instance.Init != "" {
		lines = append(lines, instance.Init)
	}
	lines = append(lines, s.Init...)

	instance.Init = strings.Join(lines, "\n")

	// Append tags
	for k, v := range s.Tags {
		if instance.Tags == nil {
			instance.Tags = map[string]string{}
		}
		instance.Tags[k] = v
	}
	return instance, nil
}
