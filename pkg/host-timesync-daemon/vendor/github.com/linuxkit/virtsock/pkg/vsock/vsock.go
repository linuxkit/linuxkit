// Package vsock provides the Linux guest bindings to VM sockets. VM
// sockets are a generic mechanism for guest<->host communication. It
// was originally developed for VMware but now also supports virtio
// sockets and (soon) Hyper-V sockets.
//
// The main purpose is to provide bindings to the Linux implementation
// of VM sockets, based on the low level support in
// golang.org/x/sys/unix.
//
// The package also provides bindings to the host interface to virtio
// sockets for HyperKit on macOS.
package vsock

import (
	"fmt"
	"net"
	"os"
)

const (
	// CIDAny is a wildcard CID
	CIDAny = 4294967295 // 2^32-1
	// CIDHypervisor is the reserved CID for the Hypervisor
	CIDHypervisor = 0
	// CIDHost is the reserved CID for the host system
	CIDHost = 2
)

// Addr represents the address of a vsock end point.
type Addr struct {
	CID  uint32
	Port uint32
}

// Network returns the network type for a Addr
func (a Addr) Network() string {
	return "vsock"
}

// String returns a string representation of a Addr
func (a Addr) String() string {
	return fmt.Sprintf("%08x.%08x", a.CID, a.Port)
}

// Conn is a vsock connection which supports half-close.
type Conn interface {
	net.Conn
	CloseRead() error
	CloseWrite() error
	File() (*os.File, error)
}
