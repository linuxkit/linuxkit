package main

import (
	"bufio"
	"io"
	"log"
	"net"
	"os"
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
		case logCh <- logEntry{Time: time.Now(), Source: "memlogd", Msg: "hello TestNonblock"}:
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
		logCh <- logEntry{Time: time.Now(), Source: "memlogd", Msg: "hello TestFinite"}
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
		logCh <- logEntry{Time: time.Now(), Source: "memlogd", Msg: "hello TestFinite2"}
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
