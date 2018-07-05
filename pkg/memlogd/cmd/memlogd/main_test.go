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
	a, b := socketpair()
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

	a, b := socketpair()
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

func socketpair() (net.Conn, net.Conn) {
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		log.Fatal("Unable to create socketpair: ", err)
	}
	f := os.NewFile(uintptr(fds[0]), "a")
	a, err := net.FileConn(f)
	if err != nil {
		log.Fatal("Unable to create net.Conn from socketpair: ", err)
	}
	_ = f.Close()
	f = os.NewFile(uintptr(fds[1]), "b")
	b, err := net.FileConn(f)
	if err != nil {
		log.Fatal("Unable to create net.Conn from socketpair: ", err)
	}
	_ = f.Close()
	return a, b
}
