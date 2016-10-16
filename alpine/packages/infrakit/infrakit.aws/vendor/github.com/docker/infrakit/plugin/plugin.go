package plugin

import (
	"io"
)

// Endpoint models some endpoint for a plugin to deliver a message to and can get raw bytes
// as response, and consequently, a typed result
// Right now one endpoint type is avalable: plugin.util.HttpEndpoint() that works http client
// and servers.
type Endpoint interface {
	// String returns a human readable representation of what this endpoint is
	String() string
}

// Handler is a server side handler of the plugin
type Handler func(vars map[string]string, body io.Reader) (result interface{}, err error)

// Callable makes something callable in a rpc context
type Callable interface {

	// String returns a string representation of the callable
	String() string

	// Call makes a call to the plugin using http method, at op (endpoint), with message and result structs
	Call(endpoint Endpoint, message, result interface{}) (raw []byte, err error)
}
