package rpc

import (
	"net/http"

	"github.com/docker/infrakit/pkg/spi"
)

// ImplementsRequest is the rpc wrapper for the Implements method args.
type ImplementsRequest struct {
}

// ImplementsResponse is the rpc wrapper for the Implements return value.
type ImplementsResponse struct {
	APIs []spi.InterfaceSpec
}

// Handshake is a simple RPC object for doing handshake
type Handshake []spi.InterfaceSpec

// Implements responds to a request for the supported plugin interfaces.
func (h Handshake) Implements(_ *http.Request, req *ImplementsRequest, resp *ImplementsResponse) error {
	resp.APIs = []spi.InterfaceSpec(h)
	return nil
}
