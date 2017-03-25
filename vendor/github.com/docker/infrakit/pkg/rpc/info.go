package rpc

const (
	// URLAPI is the well-known HTTP GET endpoint that retrieves description of the plugin's interfaces.
	URLAPI = "/info/api.json"

	// URLFunctions exposes the templates functions that are available via this plugin
	URLFunctions = "/info/functions.json"

	// URLEventsPrefix is the prefix of the events endpoint
	URLEventsPrefix = "/events"
)

// InputExample is the interface implemented by the rpc implementations for
// group, instance, and flavor to set example input using custom/ vendored data types.
type InputExample interface {

	// SetExampleProperties updates the parameter with example properties.
	// The request param must be a pointer
	SetExampleProperties(request interface{})
}
