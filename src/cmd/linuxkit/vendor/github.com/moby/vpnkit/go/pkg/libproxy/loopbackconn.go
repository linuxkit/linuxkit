package libproxy

import (
	"io"
	"net"
	"sync"
	"time"
)

// io.Pipe is synchronous but we need to decouple the Read and Write calls
// with buffering. Adding bufio.NewWriter still requires domeone else to call
// `Flush` in a background thread to perform the write. It's simpler to create
// our own bufferedPipe out of an array of []byte

// - each direction within the connection is represented by a bufferedPipe
// - each bufferedPipe can be shutdown such that further writes return EOF
//   and reads return EOF after the buffer is exhausted

type bufferedPipe struct {
	bufs         [][]byte
	eof          bool
	m            *sync.Mutex
	c            *sync.Cond
	readDeadline time.Time
}

func newBufferedPipe() *bufferedPipe {
	var m sync.Mutex
	c := sync.NewCond(&m)
	return &bufferedPipe{
		m: &m,
		c: c,
	}
}

func (pipe *bufferedPipe) TryReadLocked(p []byte) (n int, err error) {
	// drain buffers before considering EOF
	if len(pipe.bufs) > 0 {
		n := copy(p, pipe.bufs[0])
		pipe.bufs[0] = pipe.bufs[0][n:]

		if len(pipe.bufs[0]) == 0 {
			// first fragment consumed
			pipe.bufs = pipe.bufs[1:]
		}

		return n, nil
	}
	if pipe.eof {
		return 0, io.EOF
	}
	return 0, nil
}

func (pipe *bufferedPipe) SetReadDeadline(deadline time.Time) error {
	pipe.m.Lock()
	defer pipe.m.Unlock()
	pipe.readDeadline = deadline
	pipe.c.Broadcast()
	return nil
}

type errTimeout struct {
}

func (e *errTimeout) String() string {
	return "i/o timeout"
}

func (e *errTimeout) Error() string {
	return e.String()
}

func (e *errTimeout) Timeout() bool {
	return true
}

func (e *errTimeout) Temporary() bool {
	return true
}

func (pipe *bufferedPipe) Read(p []byte) (n int, err error) {
	pipe.m.Lock()
	defer pipe.m.Unlock()
	for {
		n, err := pipe.TryReadLocked(p)
		if n > 0 || err != nil {
			return n, err
		}
		done := make(chan struct{})
		timeout := make(chan time.Time)
		if !pipe.readDeadline.IsZero() {
			go func() {
				time.Sleep(time.Until(pipe.readDeadline))
				close(timeout)
			}()
		}
		go func() {
			pipe.c.Wait()
			close(done)
		}()
		select {
		case <-timeout:
			// Clean up the goroutine
			pipe.c.Broadcast()
			<-done
			return n, &errTimeout{}
		case <-done:
			// The timeout will still fire in the background
			continue
		}
	}
}

func (pipe *bufferedPipe) Write(p []byte) (n int, err error) {
	buf := make([]byte, len(p))
	copy(buf, p)
	pipe.m.Lock()
	defer pipe.m.Unlock()
	if pipe.eof {
		return 0, io.EOF
	}
	if len(p) == 0 {
		return 0, nil
	}
	pipe.bufs = append(pipe.bufs, buf)
	pipe.c.Broadcast()
	return len(p), nil
}

func (pipe *bufferedPipe) closeWriteNoErr() {
	pipe.m.Lock()
	defer pipe.m.Unlock()
	pipe.eof = true
	pipe.c.Broadcast()
}

func (pipe *bufferedPipe) CloseWrite() error {
	pipe.closeWriteNoErr()
	return nil
}

type loopback struct {
	write           *bufferedPipe
	read            *bufferedPipe
	simulateLatency time.Duration
}

func newLoopback() *loopback {
	write := newBufferedPipe()
	read := newBufferedPipe()
	return &loopback{
		write: write,
		read:  read,
	}
}

func (l *loopback) LocalAddr() net.Addr {
	return &addrLoopback{}
}

func (l *loopback) RemoteAddr() net.Addr {
	return &addrLoopback{}
}

type addrLoopback struct {
}

func (a *addrLoopback) Network() string {
	return "loopback"
}
func (a *addrLoopback) String() string {
	return ""
}

func (l *loopback) SetReadDeadline(timeout time.Time) error {
	return l.read.SetReadDeadline(timeout)
}

func (l *loopback) SetWriteDeadline(_ time.Time) error {
	// Writes never block
	return nil
}

func (l *loopback) SetDeadline(timeout time.Time) error {
	if err := l.SetReadDeadline(timeout); err != nil {
		return err
	}
	return l.SetWriteDeadline(timeout)
}

func (l *loopback) OtherEnd() *loopback {
	return &loopback{
		write: l.read,
		read:  l.write,
	}
}

func (l *loopback) Read(p []byte) (n int, err error) {
	return l.read.Read(p)
}

func (l *loopback) Write(p []byte) (n int, err error) {
	n, err = l.write.Write(p)
	time.Sleep(l.simulateLatency)
	return
}

func (l *loopback) CloseRead() error {
	return l.read.CloseWrite()
}

func (l *loopback) CloseWrite() error {
	return l.write.CloseWrite()
}

func (l *loopback) Close() error {
	err1 := l.CloseRead()
	err2 := l.CloseWrite()
	if err1 != nil {
		return err1
	}
	return err2
}

var _ Conn = &loopback{}
