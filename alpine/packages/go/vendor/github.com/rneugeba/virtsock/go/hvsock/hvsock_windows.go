package hvsock

import (
	"errors"
	"io"
	"log"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// Make sure Winsock2 is initialised
func init() {
	e := syscall.WSAStartup(uint32(0x202), &wsaData)
	if e != nil {
		log.Fatal("WSAStartup", e)
	}
}

const (
	AF_HYPERV     = 34
	SHV_PROTO_RAW = 1
	socket_error  = uintptr(^uint32(0))
)

// struck sockaddr equivalent
type rawSockaddrHyperv struct {
	Family    uint16
	Reserved  uint16
	VmId      GUID
	ServiceId GUID
}

type hvsockListener struct {
	accept_fd syscall.Handle
	laddr     HypervAddr
}

// Internal representation. Complex mostly due to asynch send()/recv() syscalls.
type hvsockConn struct {
	fd     syscall.Handle
	local  HypervAddr
	remote HypervAddr

	wg            sync.WaitGroup
	closing       bool
	readDeadline  time.Time
	writeDeadline time.Time
}

// Used for async system calls
const (
	cFILE_SKIP_COMPLETION_PORT_ON_SUCCESS = 1
	cFILE_SKIP_SET_EVENT_ON_HANDLE        = 2
)

var (
	errTimeout = &timeoutError{}

	wsaData syscall.WSAData
)

type timeoutError struct{}

func (e *timeoutError) Error() string   { return "i/o timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }

// Main constructor
func newHVsockConn(h syscall.Handle, local HypervAddr, remote HypervAddr) (*HVsockConn, error) {
	ioInitOnce.Do(initIo)
	v := &hvsockConn{fd: h, local: local, remote: remote}

	_, err := createIoCompletionPort(h, ioCompletionPort, 0, 0xffffffff)
	if err != nil {
		return nil, err
	}
	err = setFileCompletionNotificationModes(h,
		cFILE_SKIP_COMPLETION_PORT_ON_SUCCESS|cFILE_SKIP_SET_EVENT_ON_HANDLE)
	if err != nil {
		return nil, err
	}

	return &HVsockConn{hvsockConn: *v}, nil
}

// Utility function to build a struct sockaddr for syscalls.
func (a HypervAddr) sockaddr(sa *rawSockaddrHyperv) (unsafe.Pointer, int32, error) {
	sa.Family = AF_HYPERV
	sa.Reserved = 0
	for i := 0; i < len(sa.VmId); i++ {
		sa.VmId[i] = a.VmId[i]
	}
	for i := 0; i < len(sa.ServiceId); i++ {
		sa.ServiceId[i] = a.ServiceId[i]
	}

	return unsafe.Pointer(sa), int32(unsafe.Sizeof(*sa)), nil
}

func connect(s syscall.Handle, a *HypervAddr) (err error) {
	var sa rawSockaddrHyperv
	ptr, n, err := a.sockaddr(&sa)
	if err != nil {
		return err
	}

	return sys_connect(s, ptr, n)
}

func bind(s syscall.Handle, a HypervAddr) error {
	var sa rawSockaddrHyperv
	ptr, n, err := a.sockaddr(&sa)
	if err != nil {
		return err
	}

	return sys_bind(s, ptr, n)
}

func accept(s syscall.Handle, a *HypervAddr) (syscall.Handle, error) {
	return 0, errors.New("accept(): Unimplemented")
}

//
// File IO/Socket interface
//
func (s *HVsockConn) close() error {
	s.closeHandle()

	return nil
}

// Underlying raw read() function.
func (v *HVsockConn) read(buf []byte) (int, error) {
	var b syscall.WSABuf
	var bytes uint32
	var f uint32

	b.Len = uint32(len(buf))
	b.Buf = &buf[0]

	c, err := v.prepareIo()
	if err != nil {
		return 0, err
	}

	err = syscall.WSARecv(v.fd, &b, 1, &bytes, &f, &c.o, nil)
	n, err := v.asyncIo(c, v.readDeadline, bytes, err)

	// Handle EOF conditions.
	if err == nil && n == 0 && len(buf) != 0 {
		return 0, io.EOF
	}
	if err == syscall.ERROR_BROKEN_PIPE {
		return 0, io.EOF
	}

	return n, err
}

// Underlying raw write() function.
func (v *HVsockConn) write(buf []byte) (int, error) {
	var b syscall.WSABuf
	var f uint32
	var bytes uint32

	if len(buf) == 0 {
		return 0, nil
	}

	f = 0
	b.Len = uint32(len(buf))
	b.Buf = &buf[0]

	c, err := v.prepareIo()
	if err != nil {
		return 0, err
	}
	err = syscall.WSASend(v.fd, &b, 1, &bytes, f, &c.o, nil)
	return v.asyncIo(c, v.writeDeadline, bytes, err)
}

func (v *HVsockConn) SetReadDeadline(t time.Time) error {
	v.readDeadline = t
	return nil
}

func (v *HVsockConn) SetWriteDeadline(t time.Time) error {
	v.writeDeadline = t
	return nil
}

func (v *HVsockConn) SetDeadline(t time.Time) error {
	v.SetReadDeadline(t)
	v.SetWriteDeadline(t)
	return nil
}

// The code below here is adjusted from:
// https://github.com/Microsoft/go-winio/blob/master/file.go

var ioInitOnce sync.Once
var ioCompletionPort syscall.Handle

// ioResult contains the result of an asynchronous IO operation
type ioResult struct {
	bytes uint32
	err   error
}

type ioOperation struct {
	o  syscall.Overlapped
	ch chan ioResult
}

func initIo() {
	h, err := createIoCompletionPort(syscall.InvalidHandle, 0, 0, 0xffffffff)
	if err != nil {
		panic(err)
	}
	ioCompletionPort = h
	go ioCompletionProcessor(h)
}

func (v *hvsockConn) closeHandle() {
	if !v.closing {
		// cancel all IO and wait for it to complete
		v.closing = true
		cancelIoEx(v.fd, nil)
		v.wg.Wait()
		// at this point, no new IO can start
		syscall.Close(v.fd)
		v.fd = 0
	}
}

// prepareIo prepares for a new IO operation
func (s *hvsockConn) prepareIo() (*ioOperation, error) {
	s.wg.Add(1)
	if s.closing {
		return nil, ErrSocketClosed
	}
	c := &ioOperation{}
	c.ch = make(chan ioResult)
	return c, nil
}

// ioCompletionProcessor processes completed async IOs forever
func ioCompletionProcessor(h syscall.Handle) {
	// Set the timer resolution to 1. This fixes a performance regression in golang 1.6.
	timeBeginPeriod(1)
	for {
		var bytes uint32
		var key uintptr
		var op *ioOperation
		err := getQueuedCompletionStatus(h, &bytes, &key, &op, syscall.INFINITE)
		if op == nil {
			panic(err)
		}
		op.ch <- ioResult{bytes, err}
	}
}

// asyncIo processes the return value from ReadFile or WriteFile, blocking until
// the operation has actually completed.
func (v *hvsockConn) asyncIo(c *ioOperation, deadline time.Time, bytes uint32, err error) (int, error) {
	if err != syscall.ERROR_IO_PENDING {
		v.wg.Done()
		return int(bytes), err
	}

	var r ioResult
	wait := true
	timedout := false
	if v.closing {
		cancelIoEx(v.fd, &c.o)
	} else if !deadline.IsZero() {
		now := time.Now()
		if !deadline.After(now) {
			timedout = true
		} else {
			timeout := time.After(deadline.Sub(now))
			select {
			case r = <-c.ch:
				wait = false
			case <-timeout:
				timedout = true
			}
		}
	}
	if timedout {
		cancelIoEx(v.fd, &c.o)
	}
	if wait {
		r = <-c.ch
	}
	err = r.err
	if err == syscall.ERROR_OPERATION_ABORTED {
		if v.closing {
			err = ErrSocketClosed
		} else if timedout {
			err = errTimeout
		}
	}
	v.wg.Done()
	return int(r.bytes), err
}
