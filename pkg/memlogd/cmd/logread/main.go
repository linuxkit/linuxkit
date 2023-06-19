package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

const (
	logDump byte = iota
	logFollow
	logDumpFollow
)

type logEntry struct {
	Time   time.Time `json:"time"`
	Source string    `json:"source"`
	Msg    string    `json:"msg"`
}

func (msg *logEntry) String() string {
	return fmt.Sprintf("%s;%s;%s", msg.Time.Format(time.RFC3339Nano), strings.ReplaceAll(msg.Source, `;`, `\;`), msg.Msg)
}

func main() {
	var err error

	var socketPath string
	var follow bool
	var dumpFollow bool

	flag.StringVar(&socketPath, "socket", "/var/run/memlogdq.sock", "memlogd log query socket")
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

	var entry logEntry
	decoder := json.NewDecoder(conn)
	for {
		if err := decoder.Decode(&entry); err != nil {
			if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
				return
			}
			panic(err)
		}

		fmt.Println(entry.String())
	}
}
