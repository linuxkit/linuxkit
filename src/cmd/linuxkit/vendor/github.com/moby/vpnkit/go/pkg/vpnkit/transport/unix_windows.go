package transport

import (
	"context"
	"net"
	"time"

	"github.com/Microsoft/go-winio"
)

func NewUnixTransport() Transport {
	return &unix{}
}

type unix struct {
}

func (_ *unix) Dial(_ context.Context, path string) (net.Conn, error) {
	timeout := 120 * time.Second
	return winio.DialPipe(path, &timeout)
}

func (_ *unix) Listen(path string) (net.Listener, error) {
	return winio.ListenPipe(path, &winio.PipeConfig{
		MessageMode:      true,  // Use message mode so that CloseWrite() is supported
		InputBufferSize:  65536, // Use 64KB buffers to improve performance
		OutputBufferSize: 65536,
	})
}

func (_ *unix) String() string {
	return "Windows named pipe"
}
