package event

import (
	"github.com/docker/infrakit/pkg/spi"
	"github.com/docker/infrakit/pkg/types"
)

// InterfaceSpec is the current name and version of the Flavor API.
var InterfaceSpec = spi.InterfaceSpec{
	Name:    "Event",
	Version: "0.1.0",
}

// Plugin must be implemented for the object to be able to publish events.
type Plugin interface {

	// List returns a list of *child nodes* given a path for a topic.
	// A topic of "." is the top level
	List(topic types.Path) (child []string, err error)
}

// Validator is the interface for validating the topic
type Validator interface {

	// Validate validates the topic
	Validate(topic types.Path) error
}

// Publisher is the interface that event sources also implement to be assigned
// a publish function.
type Publisher interface {

	// PublishOn sets the channel to publish
	PublishOn(chan<- *Event)
}

// Subscriber is the interface given to clients interested in events
type Subscriber interface {

	// SubscribeOn returns the channel for the topic
	SubscribeOn(topic types.Path) (<-chan *Event, chan<- struct{}, error)
}
