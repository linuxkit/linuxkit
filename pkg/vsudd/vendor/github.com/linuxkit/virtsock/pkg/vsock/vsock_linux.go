// Bindings to the Linux hues interface to VM sockets.
package vsock

import (
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// SocketMode is a NOOP on Linux
func SocketMode(m string) {
}

// VsockAddr represents the address of a vsock end point.
type VsockAddr struct {
	CID  uint32
	Port uint32
}

// Network returns the network type for a VsockAddr
func (a VsockAddr) Network() string {
	return "vsock"
}

// String returns a string representation of a VsockAddr
func (a VsockAddr) String() string {
	return fmt.Sprintf("%08x.%08x", a.CID, a.Port)
}

// Convert a generic unix.Sockaddr to a VsockAddr
func sockaddrToVsock(sa unix.Sockaddr) *VsockAddr {
	switch sa := sa.(type) {
	case *unix.SockaddrVM:
		return &VsockAddr{CID: sa.CID, Port: sa.Port}
	}
	return nil
}

// Dial connects to the CID.Port via virtio sockets
func Dial(cid, port uint32) (Conn, error) {
	fd, err := syscall.Socket(unix.AF_VSOCK, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, err
	}
	sa := &unix.SockaddrVM{CID: cid, Port: port}
	if err = unix.Connect(fd, sa); err != nil {
		return nil, errors.New(fmt.Sprintf(
			"failed connect() to %08x.%08x: %s", cid, port, err))
	}
	return newVsockConn(uintptr(fd), nil, &VsockAddr{cid, port}), nil
}

// Listen returns a net.Listener which can accept connections on the given port
func Listen(cid, port uint32) (net.Listener, error) {
	fd, err := syscall.Socket(unix.AF_VSOCK, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, err
	}

	sa := &unix.SockaddrVM{CID: cid, Port: port}
	if err = unix.Bind(fd, sa); err != nil {
		return nil, errors.New(fmt.Sprintf(
			"failed bind() to %08x.%08x: %s", cid, port, err))
	}

	err = syscall.Listen(fd, syscall.SOMAXCONN)
	if err != nil {
		return nil, err
	}
	return &vsockListener{fd, VsockAddr{cid, port}}, nil
}

type vsockListener struct {
	fd    int
	local VsockAddr
}

// Accept accepts an incoming call and returns the new connection.
func (v *vsockListener) Accept() (net.Conn, error) {
	fd, sa, err := unix.Accept(v.fd)
	if err != nil {
		return nil, err
	}
	return newVsockConn(uintptr(fd), &v.local, sockaddrToVsock(sa)), nil
}

// Close closes the listening connection
func (v *vsockListener) Close() error {
	// Note this won't cause the Accept to unblock.
	return unix.Close(v.fd)
}

// Addr returns the address the Listener is listening on
func (v *vsockListener) Addr() net.Addr {
	return v.local
}

// a wrapper around FileConn which supports CloseRead and CloseWrite
type vsockConn struct {
	vsock  *os.File
	fd     uintptr
	local  *VsockAddr
	remote *VsockAddr
}

// VsockConn represents a connection over a vsock
type VsockConn struct {
	vsockConn
}

func newVsockConn(fd uintptr, local, remote *VsockAddr) *VsockConn {
	vsock := os.NewFile(fd, fmt.Sprintf("vsock:%d", fd))
	return &VsockConn{vsockConn{vsock: vsock, fd: fd, local: local, remote: remote}}
}

// LocalAddr returns the local address of a connection
func (v *VsockConn) LocalAddr() net.Addr {
	return v.local
}

// RemoteAddr returns the remote address of a connection
func (v *VsockConn) RemoteAddr() net.Addr {
	return v.remote
}

// Close closes the connection
func (v *VsockConn) Close() error {
	return v.vsock.Close()
}

// CloseRead shuts down the reading side of a vsock connection
func (v *VsockConn) CloseRead() error {
	return syscall.Shutdown(int(v.fd), syscall.SHUT_RD)
}

// CloseWrite shuts down the writing side of a vsock connection
func (v *VsockConn) CloseWrite() error {
	return syscall.Shutdown(int(v.fd), syscall.SHUT_WR)
}

// Read reads data from the connection
func (v *VsockConn) Read(buf []byte) (int, error) {
	return v.vsock.Read(buf)
}

// Write writes data over the connection
func (v *VsockConn) Write(buf []byte) (int, error) {
	return v.vsock.Write(buf)
}

// SetDeadline sets the read and write deadlines associated with the connection
func (v *VsockConn) SetDeadline(t time.Time) error {
	return nil // FIXME
}

// SetReadDeadline sets the deadline for future Read calls.
func (v *VsockConn) SetReadDeadline(t time.Time) error {
	return nil // FIXME
}

// SetWriteDeadline sets the deadline for future Write calls
func (v *VsockConn) SetWriteDeadline(t time.Time) error {
	return nil // FIXME
}
