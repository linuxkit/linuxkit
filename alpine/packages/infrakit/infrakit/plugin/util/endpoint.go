package util

import (
	"fmt"
	"github.com/docker/infrakit/plugin"
)

// HTTPEndpoint is a specialization of an endpoint. It implements the Endpoint interface
type HTTPEndpoint struct {
	Method string
	Path   string
}

func (h *HTTPEndpoint) String() string {
	return "http:" + h.Method + ":" + h.Path
}

// GetHTTPEndpoint returns an http endpoint if the input endpoint is a supported http endpoint.
func GetHTTPEndpoint(endpoint plugin.Endpoint) (*HTTPEndpoint, error) {
	ep, ok := endpoint.(*HTTPEndpoint)
	if !ok {
		return nil, fmt.Errorf("unsupported endpoint: %v", endpoint)
	}

	if ep.Method == "" || ep.Path == "" {
		return nil, fmt.Errorf("invalid http endpoint:%v", endpoint)
	}
	return ep, nil
}
