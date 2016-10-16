package instance

import (
	"encoding/json"
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
type Attachment string

// Spec is a specification of an instance to be provisioned
type Spec struct {
	Properties  *json.RawMessage
	Tags        map[string]string
	Init        string
	LogicalID   *LogicalID
	Attachments []Attachment
}
