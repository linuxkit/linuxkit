package transport

import (
	"context"
	"net"
)

// Transport carries the HTTP port control messages.
type Transport interface {
	Dial(_ context.Context, path string) (net.Conn, error)
	Listen(path string) (net.Listener, error)
	String() string
}

// Choose a transport based on a path.
func Choose(path string) Transport {
	_, err := parseAddr(path)
	if err == nil {
		return NewVsockTransport()
	}
	return NewUnixTransport()
}
