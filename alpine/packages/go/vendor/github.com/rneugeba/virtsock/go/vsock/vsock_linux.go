package vsock

import (
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"
	"time"
)

/* No way to teach net or syscall about vsock sockaddr, so go right to C */

/*
#include <sys/socket.h>

struct sockaddr_vm {
	sa_family_t svm_family;
	unsigned short svm_reserved1;
	unsigned int svm_port;
	unsigned int svm_cid;
	unsigned char svm_zero[sizeof(struct sockaddr) -
		sizeof(sa_family_t) - sizeof(unsigned short) -
		sizeof(unsigned int) - sizeof(unsigned int)];
};

int bind_sockaddr_vm(int fd, const struct sockaddr_vm *sa_vm) {
    return bind(fd, (const struct sockaddr*)sa_vm, sizeof(*sa_vm));
}
int connect_sockaddr_vm(int fd, const struct sockaddr_vm *sa_vm) {
    return connect(fd, (const struct sockaddr*)sa_vm, sizeof(*sa_vm));
}
int accept_vm(int fd, struct sockaddr_vm *sa_vm, socklen_t *sa_vm_len) {
    return accept4(fd, (struct sockaddr *)sa_vm, sa_vm_len, 0);
}
*/
import "C"

const (
	AF_VSOCK             = 40
	VSOCK_CID_ANY        = 4294967295 /* 2^32-1 */
	VSOCK_CID_HYPERVISOR = 0
	VSOCK_CID_HOST       = 2
	VSOCK_CID_SELF       = 3
)

func Dial(cid, port uint) (Conn, error) {
	fd, err := syscall.Socket(AF_VSOCK, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, err
	}

	sa := C.struct_sockaddr_vm{}
	sa.svm_family = AF_VSOCK
	sa.svm_port = C.uint(port)
	sa.svm_cid = C.uint(cid)

	if ret, errno := C.connect_sockaddr_vm(C.int(fd), &sa); ret != 0 {
		return nil, errors.New(fmt.Sprintf(
			"failed bind connect to %08x.%08x, returned %d, errno %d: %s",
			sa.svm_cid, sa.svm_port, ret, errno, errno))
	}

	return newVsockConn(uintptr(fd), port)
}

// Listen returns a net.Listener which can accept connections on the given
// vhan port.
func Listen(port uint) (net.Listener, error) {
	accept_fd, err := syscall.Socket(AF_VSOCK, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, err
	}

	sa := C.struct_sockaddr_vm{}
	sa.svm_family = AF_VSOCK
	sa.svm_port = C.uint(port)
	sa.svm_cid = VSOCK_CID_ANY

	if ret, errno := C.bind_sockaddr_vm(C.int(accept_fd), &sa); ret != 0 {
		return nil, errors.New(fmt.Sprintf(
			"failed bind vsock connection to %08x.%08x, returned %d, errno %d: %s",
			sa.svm_cid, sa.svm_port, ret, errno, errno))
	}

	err = syscall.Listen(accept_fd, syscall.SOMAXCONN)
	if err != nil {
		return nil, err
	}
	return &vsockListener{accept_fd, port}, nil
}

// Conn is a vsock connection which support half-close.
type Conn interface {
	net.Conn
	CloseRead() error
	CloseWrite() error
}

type vsockListener struct {
	accept_fd int
	port      uint
}

func (v *vsockListener) Accept() (net.Conn, error) {
	var accept_sa C.struct_sockaddr_vm
	var accept_sa_len C.socklen_t

	accept_sa_len = C.sizeof_struct_sockaddr_vm
	fd, err := C.accept_vm(C.int(v.accept_fd), &accept_sa, &accept_sa_len)
	if err != nil {
		return nil, err
	}
	return newVsockConn(uintptr(fd), v.port)
}

func (v *vsockListener) Close() error {
	// Note this won't cause the Accept to unblock.
	return syscall.Close(v.accept_fd)
}

type VsockAddr struct {
	Port uint
}

func (a VsockAddr) Network() string {
	return "vsock"
}

func (a VsockAddr) String() string {
	return fmt.Sprintf("%08x", a.Port)
}

func (v *vsockListener) Addr() net.Addr {
	return VsockAddr{Port: v.port}
}

// a wrapper around FileConn which supports CloseRead and CloseWrite
type vsockConn struct {
	vsock  *os.File
	fd     uintptr
	local  VsockAddr
	remote VsockAddr
}

type VsockConn struct {
	vsockConn
}

func newVsockConn(fd uintptr, localPort uint) (*VsockConn, error) {
	vsock := os.NewFile(fd, fmt.Sprintf("vsock:%d", fd))
	local := VsockAddr{Port: localPort}
	remote := VsockAddr{Port: uint(0)} // FIXME
	return &VsockConn{vsockConn{vsock: vsock, fd: fd, local: local, remote: remote}}, nil
}

func (v *VsockConn) LocalAddr() net.Addr {
	return v.local
}

func (v *VsockConn) RemoteAddr() net.Addr {
	return v.remote
}

func (v *VsockConn) CloseRead() error {
	return syscall.Shutdown(int(v.fd), syscall.SHUT_RD)
}

func (v *VsockConn) CloseWrite() error {
	return syscall.Shutdown(int(v.fd), syscall.SHUT_WR)
}

func (v *VsockConn) Close() error {
	return v.vsock.Close()
}

func (v *VsockConn) Read(buf []byte) (int, error) {
	return v.vsock.Read(buf)
}

func (v *VsockConn) Write(buf []byte) (int, error) {
	return v.vsock.Write(buf)
}

func (v *VsockConn) SetDeadline(t time.Time) error {
	return nil // FIXME
}

func (v *VsockConn) SetReadDeadline(t time.Time) error {
	return nil // FIXME
}

func (v *VsockConn) SetWriteDeadline(t time.Time) error {
	return nil // FIXME
}
