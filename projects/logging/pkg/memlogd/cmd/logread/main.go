package main

import (
	"bufio"
	"flag"
	"net"
	"os"
)

const (
	logDump byte = iota
	logFollow
	logDumpFollow
)

func main() {
	var err error

	var socketPath string
	var follow bool
	var dumpFollow bool

	flag.StringVar(&socketPath, "socket", "/tmp/memlogdq.sock", "memlogd log query socket")
	flag.BoolVar(&dumpFollow, "F", false, "dump log, then follow")
	flag.BoolVar(&follow, "f", false, "follow log buffer")
	flag.Parse()

	addr := net.UnixAddr{
		Name: socketPath,
		Net:  "unix",
	}
	conn, err := net.DialUnix("unix", nil, &addr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	var n int
	switch {
	case dumpFollow:
		n, err = conn.Write([]byte{logDumpFollow})
	case follow && !dumpFollow:
		n, err = conn.Write([]byte{logFollow})
	default:
		n, err = conn.Write([]byte{logDump})
	}

	if err != nil || n < 1 {
		panic(err)
	}

	r := bufio.NewReader(conn)
	r.WriteTo(os.Stdout)

}
