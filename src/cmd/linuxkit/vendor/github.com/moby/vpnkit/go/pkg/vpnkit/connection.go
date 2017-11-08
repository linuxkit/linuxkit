package vpnkit

import (
	"context"

	datakit "github.com/moby/datakit/api/go-datakit"
)

// Connection represents an open control connection to vpnkit
type Connection struct {
	client *datakit.Client
}

// NewConnection connects to a vpnkit Unix domain socket on the given path
// and returns the connection
func NewConnection(ctx context.Context, path string) (*Connection, error) {
	client, err := datakit.Dial(ctx, "unix", path)
	if err != nil {
		return nil, err
	}
	return &Connection{client}, nil
}
