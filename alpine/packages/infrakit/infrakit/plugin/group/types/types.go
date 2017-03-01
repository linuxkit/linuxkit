package types

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/docker/infrakit/spi/group"
	"github.com/docker/infrakit/spi/instance"
)

// Spec is the configuration schema for the plugin, provided in group.Spec.Properties
type Spec struct {
	Instance   InstancePlugin
	Flavor     FlavorPlugin
	Allocation AllocationMethod
}

// AllocationMethod defines the type of allocation and supervision needed by a flavor's Group.
type AllocationMethod struct {
	Size       uint
	LogicalIDs []instance.LogicalID
}

// InstancePlugin is the structure that describes an instance plugin.
type InstancePlugin struct {
	Plugin     string
	Properties *json.RawMessage // this will be the Spec of the plugin
}

// FlavorPlugin describes the flavor configuration
type FlavorPlugin struct {
	Plugin     string
	Properties *json.RawMessage // this will be the Spec of the plugin
}

// ParseProperties parses the group plugin properties JSON document in a group configuration.
func ParseProperties(config group.Spec) (Spec, error) {
	parsed := Spec{}
	if err := json.Unmarshal([]byte(RawMessage(config.Properties)), &parsed); err != nil {
		return parsed, fmt.Errorf("Invalid properties: %s", err)
	}
	return parsed, nil
}

// MustParse can be wrapped over ParseProperties to panic if parsing fails.
func MustParse(s Spec, e error) Spec {
	if e != nil {
		panic(e)
	}
	return s
}

func stableFormat(v interface{}) []byte {
	// Marshal the JSON to ensure stable key ordering.  This allows structurally-identical JSON to yield the same
	// hash even if the fields are reordered.

	unstable, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}

	props := map[string]interface{}{}
	err = json.Unmarshal(unstable, &props)
	if err != nil {
		panic(err)
	}

	stable, err := json.MarshalIndent(props, "  ", "  ") // sorts the fields
	if err != nil {
		panic(err)
	}
	return stable
}

// InstanceHash computes a stable hash of the document in InstancePluginProperties.
func (c Spec) InstanceHash() string {
	// TODO(wfarner): This does not consider changes made by plugins that are not represented by user
	// configuration changes, such as if a plugin is updated.  We may be able to address this by resolving plugin
	// names to a versioned plugin identifier.

	hasher := sha1.New()
	hasher.Write(stableFormat(c.Instance))
	hasher.Write(stableFormat(c.Flavor))
	return base64.URLEncoding.EncodeToString(hasher.Sum(nil))
}

// RawMessage converts a pointer to a raw message to a copy of the value. If the pointer is nil, it returns
// an empty raw message.  This is useful for structs where fields are json.RawMessage pointers for bi-directional
// marshal and unmarshal (value receivers will encode base64 instead of raw json when marshaled), so bi-directional
// structs should use pointer fields.
func RawMessage(r *json.RawMessage) (raw json.RawMessage) {
	if r != nil {
		raw = *r
	}
	return
}
