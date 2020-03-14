// +build !windows

package transport

import (
	"context"
	"net"
	"os"

	"github.com/pkg/errors"
)

func NewUnixTransport() Transport {
	return &unix{}
}

type unix struct {
}

func (_ *unix) Dial(_ context.Context, path string) (net.Conn, error) {
	return net.Dial("unix", path)
}

func (_ *unix) Listen(path string) (net.Listener, error) {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return nil, errors.Wrap(err, "removing "+path)
	}
	return net.Listen("unix", path)
}

func (_ *unix) String() string {
	return "Unix domain socket"
}
