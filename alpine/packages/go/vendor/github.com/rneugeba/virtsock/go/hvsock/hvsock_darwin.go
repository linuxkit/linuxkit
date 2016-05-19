// Dummy implementation to compile on Mac OSX

package hvsock
import (
	"errors"
	"time"
)

const (
	AF_HYPERV     = 42
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
	return errors.New("connect() not implemented")
}

func bind(s int, a HypervAddr) error {
	return errors.New("bind() not implemented")
}

func accept(s int, a *HypervAddr) (int, error) {
	return 0, errors.New("accept() not implemented")
}

// Internal representation. Complex mostly due to asynch send()/recv() syscalls.
type hvsockConn struct {
	fd     int
	local  HypervAddr
	remote HypervAddr
}

// Main constructor
func newHVsockConn(fd int, local HypervAddr, remote HypervAddr) (*HVsockConn, error) {
	v := &hvsockConn{local: local, remote: remote}
	return &HVsockConn{hvsockConn: *v}, errors.New("newHVsockConn() not implemented")
}

func (v *HVsockConn) close() error {
	return errors.New("close() not implemented")
}

func (v *HVsockConn) read(buf []byte) (int, error) {
	return 0, errors.New("read() not implemented")
}

func (v *HVsockConn) write(buf []byte) (int, error) {
	return 0, errors.New("write() not implemented")
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
