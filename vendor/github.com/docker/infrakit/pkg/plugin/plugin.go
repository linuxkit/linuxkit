package plugin

import (
	"github.com/docker/infrakit/pkg/spi"
	"github.com/docker/infrakit/pkg/template"
	"github.com/docker/infrakit/pkg/types"
)

// Spec models a canonical pattern of fields that exist in a struct/ map / union that indicates the block is a plugin.
type Spec struct {

	// Plugin is the name of the plugin
	Plugin Name

	// Properties is the configuration of the plugin
	Properties *types.Any
}

// Informer is the interface that gives information about the plugin such as version and interface methods
type Informer interface {

	// GetInfo returns metadata about the plugin
	GetInfo() (Info, error)

	// GetFunctions returns metadata about the plugin's template functions, if the plugin supports templating.
	GetFunctions() (map[string][]template.Function, error)
}

// Info is metadata for the plugin
type Info struct {

	// Vendor captures vendor-specific information about this plugin
	Vendor *spi.VendorInfo

	// Implements is a list of plugin interface and versions this plugin supports
	Implements []spi.InterfaceSpec

	// Interfaces (optional) is a slice of interface descriptions by the type and version
	Interfaces []InterfaceDescription `json:",omitempty"`
}

// InterfaceDescription is a holder for RPC interface version and method descriptions
type InterfaceDescription struct {
	spi.InterfaceSpec
	Methods []MethodDescription
}

// MethodDescription contains information about the RPC method such as the request and response
// example structs.  The request value can be used as an example input, possibly with example
// plugin-custom properties if the underlying plugin implements the InputExample interface.
// The response value gives an example of the example response.
type MethodDescription struct {
	// Request is the RPC request example
	Request Request

	// Response is the RPC response example
	Response Response
}

// Request models the RPC request payload
type Request struct {

	// Version is the version of the JSON RPC protocol
	Version string `json:"jsonrpc"`

	// Method is the rpc method to use in the payload field 'method'
	Method string `json:"method"`

	// Params contains example inputs.  This can be a zero value struct or one with defaults
	Params interface{} `json:"params"`

	// ID is the request is
	ID string `json:"id"`
}

// Response is the RPC response struct
type Response struct {

	// Result is the result of the call
	Result interface{} `json:"result"`

	// ID is id matching the request ID
	ID string `json:"id"`
}

// Endpoint is the address of the plugin service
type Endpoint struct {

	// Name is the key used to refer to this plugin in all JSON configs
	Name string

	// Protocol is the transport protocol -- unix, tcp, etc.
	Protocol string

	// Address is the how to connect - socket file, host:port, etc.
	Address string
}
