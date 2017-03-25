package instance

import (
	"github.com/docker/infrakit/pkg/types"
)

// ID is the identifier for an instance.
type ID string

// Description contains details about an instance.
type Description struct {
	ID        ID
	LogicalID *LogicalID
	Tags      map[string]string
}

// LogicalID is the logical identifier to associate with an instance.
type LogicalID string

// Attachment is an identifier for a resource to attach to an instance.
type Attachment struct {
	// ID is the unique identifier for the attachment.
	ID string

	// Type is the kind of attachment.  This allows multiple attachments of different types, with the supported
	// types defined by the plugin.
	Type string
}

// Spec is a specification of an instance to be provisioned
type Spec struct {
	// Properties is the opaque instance plugin configuration.
	Properties *types.Any

	// Tags are metadata that describes an instance.
	Tags map[string]string

	// Init is the boot script to execute when the instance is created.
	Init string

	// LogicalID is the logical identifier assigned to this instance, which may be absent.
	LogicalID *LogicalID

	// Attachments are instructions for external entities that should be attached to the instance.
	Attachments []Attachment
}
