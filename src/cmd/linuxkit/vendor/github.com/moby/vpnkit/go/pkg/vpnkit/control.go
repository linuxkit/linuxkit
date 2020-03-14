package vpnkit

import "github.com/moby/vpnkit/go/pkg/libproxy"

// Control is the port-forwarding control-plane
type Control interface {
	Mux() libproxy.Multiplexer   // Mux is the current multiplexer to forward to
	SetMux(libproxy.Multiplexer) // SetMux updates the current multiplexer for future connections
}
