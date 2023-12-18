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

// LogEntry the structure of a log entry
type LogEntry struct {
	Time   time.Time `json:"time"`
	Source string    `json:"source"`
	Msg    string    `json:"msg"`
	Error  error
}

func (msg *LogEntry) String() string {
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

	c, err := StreamLogs(socketPath, follow, dumpFollow)
	if err != nil {
		panic(err)
	}
	for entry := range c {
		if entry.Error != nil {
			panic(entry.Error)
		}
		fmt.Println(entry.String())
	}
}

// StreamLogs read the memlogd logs from socketPath, convert them to LogEntry struct
// and send those on the return channel. If there is an error in parsing, it will be the
// Error on the LogEntry struct. When the socket is closed, will close the channel.
// If stream is complete, will close, unless follow is true, in which case it will
// continue to listen for new logs.
func StreamLogs(socketPath string, follow, dump bool) (<-chan LogEntry, error) {
	addr := net.UnixAddr{
		Name: socketPath,
		Net:  "unix",
	}
	conn, err := net.DialUnix("unix", nil, &addr)
	if err != nil {
		return nil, err
	}

	var n int
	switch {
	case follow && dump:
		n, err = conn.Write([]byte{logDumpFollow})
	case follow:
		n, err = conn.Write([]byte{logFollow})
	default:
		n, err = conn.Write([]byte{logDump})
	}

	if err != nil || n < 1 {
		return nil, err
	}

	c := make(chan LogEntry)
	go func(c chan<- LogEntry) {
		defer conn.Close()
		var (
			entry   LogEntry
			decoder = json.NewDecoder(conn)
		)
		for {
			if err := decoder.Decode(&entry); err != nil {
				if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
					close(c)
					return
				}
				entry = LogEntry{Error: err}
			}
			c <- entry
		}
	}(c)
	return c, nil
}
