package hvsock

/*
#include <sys/socket.h>

struct sockaddr_hv {
	unsigned short shv_family;
	unsigned short reserved;
	unsigned char  shv_vm_id[16];
	unsigned char  shv_service_id[16];
};
int bind_sockaddr_hv(int fd, const struct sockaddr_hv *sa_hv) {
    return bind(fd, (const struct sockaddr*)sa_hv, sizeof(*sa_hv));
}
int connect_sockaddr_hv(int fd, const struct sockaddr_hv *sa_hv) {
    return connect(fd, (const struct sockaddr*)sa_hv, sizeof(*sa_hv));
}
int accept_hv(int fd, struct sockaddr_hv *sa_hv, socklen_t *sa_hv_len) {
    return accept4(fd, (struct sockaddr *)sa_hv, sa_hv_len, 0);
}
*/
import "C"

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	AF_HYPERV     = 43
	SHV_PROTO_RAW = 1
)

type hvsockListener struct {
	accept_fd int
	laddr     HypervAddr
}

//
// System call wrapper
//
func connect(s int, a *HypervAddr) (err error) {
	sa := C.struct_sockaddr_hv{}
	sa.shv_family = AF_HYPERV
	sa.reserved = 0

	for i := 0; i < 16; i++ {
		sa.shv_vm_id[i] = C.uchar(a.VmId[i])
	}
	for i := 0; i < 16; i++ {
		sa.shv_service_id[i] = C.uchar(a.ServiceId[i])
	}

	if ret := C.connect_sockaddr_hv(C.int(s), &sa); ret != 0 {
		return errors.New("connect() returned " + strconv.Itoa(int(ret)))
	}

	return nil
}

func bind(s int, a HypervAddr) error {
	sa := C.struct_sockaddr_hv{}
	sa.shv_family = AF_HYPERV
	sa.reserved = 0

	for i := 0; i < 16; i++ {
		// XXX this should take the address from `a` but Linux
		// currently only support 0s
		sa.shv_vm_id[i] = C.uchar(GUID_ZERO[i])
	}
	for i := 0; i < 16; i++ {
		sa.shv_service_id[i] = C.uchar(a.ServiceId[i])
	}

	if ret := C.bind_sockaddr_hv(C.int(s), &sa); ret != 0 {
		return errors.New("bind() returned " + strconv.Itoa(int(ret)))
	}

	return nil
}

func accept(s int, a *HypervAddr) (int, error) {
	var accept_sa C.struct_sockaddr_hv
	var accept_sa_len C.socklen_t

	accept_sa_len = C.sizeof_struct_sockaddr_hv
	fd, err := C.accept_hv(C.int(s), &accept_sa, &accept_sa_len)
	if err != nil {
		return -1, err
	}

	a.VmId = guidFromC(accept_sa.shv_vm_id)
	a.ServiceId = guidFromC(accept_sa.shv_service_id)

	return int(fd), nil
}

// Internal representation. Complex mostly due to asynch send()/recv() syscalls.
type hvsockConn struct {
	fd     int
	hvsock *os.File
	local  HypervAddr
	remote HypervAddr
}

// Main constructor
func newHVsockConn(fd int, local HypervAddr, remote HypervAddr) (*HVsockConn, error) {
	hvsock := os.NewFile(uintptr(fd), fmt.Sprintf("hvsock:%d", fd))
	v := &hvsockConn{fd: fd, hvsock: hvsock, local: local, remote: remote}

	return &HVsockConn{hvsockConn: *v}, nil
}

func (v *HVsockConn) close() error {
	return v.hvsock.Close()
}

func (v *HVsockConn) read(buf []byte) (int, error) {
	return v.hvsock.Read(buf)
}

func (v *HVsockConn) write(buf []byte) (int, error) {
	return v.hvsock.Write(buf)
}

func (v *HVsockConn) SetReadDeadline(t time.Time) error {
	return nil // FIXME
}

func (v *HVsockConn) SetWriteDeadline(t time.Time) error {
	return nil // FIXME
}

func (v *HVsockConn) SetDeadline(t time.Time) error {
	return nil // FIXME
}

func guidFromC(cg [16]C.uchar) GUID {
	var g GUID
	for i := 0; i < 16; i++ {
		g[i] = byte(cg[i])
	}
	return g
}
