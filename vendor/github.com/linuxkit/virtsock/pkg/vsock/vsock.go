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
	"net"
)

const (
	// CIDAny is a wildcard CID
	CIDAny = 4294967295 // 2^32-1
	// CIDHypervisor is the reserved CID for the Hypervisor
	CIDHypervisor = 0
	// CIDHost is the reserved CID for the host system
	CIDHost = 2
)

// Conn is a vsock connection which supports half-close.
type Conn interface {
	net.Conn
	CloseRead() error
	CloseWrite() error
}
