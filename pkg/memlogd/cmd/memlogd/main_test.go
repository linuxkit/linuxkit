package main

import (
	"bufio"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestNonblock(t *testing.T) {
	// Test that writes to the logger don't block because it is full
	linesInBuffer := 10

	logCh := make(chan logEntry)
	queryMsgChan := make(chan queryMessage)

	go ringBufferHandler(linesInBuffer, linesInBuffer, logCh, queryMsgChan)

	// Overflow the log to make sure it doesn't block
	for i := 0; i < 2*linesInBuffer; i++ {
		select {
		case logCh <- logEntry{time: time.Now(), source: "memlogd", msg: "hello TestNonblock"}:
			continue
		case <-time.After(time.Second):
			t.Errorf("write to the logger blocked for over 1s after %d (size was set to %d)", i, linesInBuffer)
		}
	}
}

func TestFinite(t *testing.T) {
	// Test that the logger doesn't store more than its configured maximum size
	linesInBuffer := 10

	logCh := make(chan logEntry)
	queryMsgChan := make(chan queryMessage)

	go ringBufferHandler(linesInBuffer, linesInBuffer, logCh, queryMsgChan)

	// Overflow the log by 2x
	for i := 0; i < 2*linesInBuffer; i++ {
		logCh <- logEntry{time: time.Now(), source: "memlogd", msg: "hello TestFinite"}
	}
	a, b := loopback()
	defer a.Close()
	defer b.Close()
	queryM := queryMessage{
		conn: a,
		mode: logDump,
	}
	queryMsgChan <- queryM
	r := bufio.NewReader(b)
	count := 0
	for {
		_, err := r.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Unexpected error reading from socket: %s", err)
		}
		count++
	}
	if linesInBuffer != count {
		t.Errorf("Read %d lines but expected %d", count, linesInBuffer)
	}
}

func TestFinite2(t *testing.T) {
	// Test that the query connection doesn't store more than the configured
	// maximum size.
	linesInBuffer := 10
	// the output buffer size will be 1/2 of the ring
	outputBufferSize := linesInBuffer / 2
	logCh := make(chan logEntry)
	queryMsgChan := make(chan queryMessage)

	go ringBufferHandler(linesInBuffer, outputBufferSize, logCh, queryMsgChan)

	// fill the ring
	for i := 0; i < linesInBuffer; i++ {
		logCh <- logEntry{time: time.Now(), source: "memlogd", msg: "hello TestFinite2"}
	}

	a, b := loopback()
	defer a.Close()
	defer b.Close()
	queryM := queryMessage{
		conn: a,
		mode: logDump,
	}
	queryMsgChan <- queryM
	// since the ring won't fit in the output buffer some should be dropped

	r := bufio.NewReader(b)
	count := 0
	for {
		_, err := r.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Unexpected error reading from socket: %s", err)
		}
		count++
	}
	if count != outputBufferSize {
		t.Errorf("Read %d lines but expected %d", count, outputBufferSize)
	}
}

func TestGoodName(t *testing.T) {
	// Test that the source names can't contain ";"
	linesInBuffer := 10
	logCh := make(chan logEntry)
	fdMsgChan := make(chan fdMessage)
	queryMsgChan := make(chan queryMessage)

	go ringBufferHandler(linesInBuffer, linesInBuffer, logCh, queryMsgChan)
	go loggingRequestHandler(80, logCh, fdMsgChan)

	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		log.Fatal("Unable to create socketpair: ", err)
	}
	a := fdToConn(fds[0])
	b := fdToConn(fds[1])
	defer a.Close()
	defer b.Close()
	// defer close fds

	fdMsgChan <- fdMessage{
		name: "semi-colons are banned;",
		fd:   fds[0],
	}
	// although the fd should be rejected my memlogd the Write should be buffered
	// by the kernel and not block.
	if _, err := b.Write([]byte("hello\n")); err != nil {
		log.Fatalf("Failed to write log message: %s", err)
	}
	c, d := loopback()
	defer c.Close()
	defer d.Close()
	// this log should not be in the ring because the connection was rejected.
	queryM := queryMessage{
		conn: c,
		mode: logDumpFollow,
	}
	queryMsgChan <- queryM
	// The error log is generated asynchronously. It should be fast. On error time out
	// after 5s.
	d.SetDeadline(time.Now().Add(5 * time.Second))
	r := bufio.NewReader(d)
	for {
		line, err := r.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Unexpected error reading from socket: %s", err)
		}
		if strings.Contains(line, "ERROR: cannot register log") {
			return
		}
	}
	t.Fatal("Failed to read error message when registering a log with a ;")
}

// caller must close fd themselves: closing the net.Conn will not close fd.
func fdToConn(fd int) net.Conn {
	f := os.NewFile(uintptr(fd), "")
	c, err := net.FileConn(f)
	if err != nil {
		log.Fatal("Unable to create net.Conn from file descriptor: ", err)
	}
	return c
}

func loopback() (net.Conn, net.Conn) {
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		log.Fatal("Unable to create socketpair: ", err)
	}
	a := fdToConn(fds[0])
	b := fdToConn(fds[1])
	// net.Conns are independent of the fds, so we must close the fds now.
	if err := syscall.Close(fds[0]); err != nil {
		log.Fatal("Unable to close socketpair fd: ", err)
	}
	if err := syscall.Close(fds[1]); err != nil {
		log.Fatal("Unable to close socketpair fd: ", err)
	}
	return a, b
}
