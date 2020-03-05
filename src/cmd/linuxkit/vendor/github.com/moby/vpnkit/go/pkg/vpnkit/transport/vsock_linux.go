package transport

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/linuxkit/virtsock/pkg/vsock"
	"github.com/pkg/errors"
)

func NewVsockTransport() Transport {
	return &vs{}
}

type vs struct {
}

func (_ *vs) Dial(_ context.Context, path string) (net.Conn, error) {
	addr, err := parseAddr(path)
	if err != nil {
		return nil, err
	}
	cid := uint32(vsock.CIDHost)
	if addr.cid != vsock.CIDAny {
		cid = addr.cid
	}
	return vsock.Dial(cid, addr.port)
}

func (_ *vs) Listen(path string) (net.Listener, error) {
	addr, err := parseAddr(path)
	if err != nil {
		return nil, err
	}
	return vsock.Listen(vsock.CIDAny, addr.port)
}

func (_ *vs) String() string {
	return "Linux AF_VSOCK"
}

type addr struct {
	cid  uint32
	port uint32
}

func parseAddr(path string) (*addr, error) {
	// The string has an optional <vm>/ prefix
	bits := strings.SplitN(path, "/", 2)
	// The last thing on the string is always the port number
	portString := bits[0]
	if len(bits) == 2 {
		portString = bits[1]
	}
	addr := &addr{
		cid:  vsock.CIDAny,
		port: 0,
	}

	port, err := strconv.ParseUint(portString, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("cannot parse %s as service GUID or AF_VSOCK port", portString)
	}
	addr.port = uint32(port)

	// Is there a <vm>/ prefix?
	if len(bits) == 1 {
		return addr, nil
	}

	// Maybe it's an integer
	cid, err := strconv.ParseUint(bits[0], 10, 32)
	if err != nil {
		return nil, errors.New("unable to parse the <vm>/ as either a GUID or AF_VSOCK port number")
	}
	addr.cid = uint32(cid)
	return addr, nil
}
