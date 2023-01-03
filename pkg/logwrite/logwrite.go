package main

// Write logs to files and perform rotation.

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// These must be kept in sync with memlogd:
const (
	logDump byte = iota
	logFollow
	logDumpFollow
)

const mb = 1024 * 1024

// LogMessage is a message received from memlogd.
type LogMessage struct {
	Time   time.Time `json:"time"`   // time message was received by memlogd
	Source string    `json:"source"` // name of the service that wrote the message
	Msg    string    `json:"msg"`    // body of the message
}

func (m *LogMessage) String() string {
	return m.Time.Format(time.RFC3339) + " " + m.Source + " " + m.Msg
}

// LogFile is where we write LogMessages to
type LogFile struct {
	File         *os.File // active file handle
	Path         string   // Path to the logfile
	BytesWritten int      // total number of bytes written so far
}

// NewLogFile creates a new LogFile.
func NewLogFile(dir, name string) (*LogFile, error) {
	// If the log exists already we want to append to it.
	p := filepath.Join(dir, name+".log")
	f, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	return &LogFile{
		File:         f,
		Path:         p,
		BytesWritten: int(fi.Size()),
	}, nil
}

// Write appends a message to the log file
func (l *LogFile) Write(m *LogMessage) error {
	s := m.String() + "\n"
	_, err := io.WriteString(l.File, s)
	if err == nil {
		l.BytesWritten += len(s)
	}
	return err
}

// Close a log file
func (l *LogFile) Close() error {
	return l.File.Close()
}

// Rotate closes the current log file, rotates the files and creates an empty log file.
func (l *LogFile) Rotate(maxLogFiles int) error {
	if err := l.File.Close(); err != nil {
		return err
	}
	for i := maxLogFiles - 1; i >= 0; i-- {
		newerFile := fmt.Sprintf("%s.%d", l.Path, i-1)
		// special case: if index is 0 we omit the suffix i.e. we expect
		// foo foo.1 foo.2 up to foo.<maxLogFiles-1>
		if i == 0 {
			newerFile = l.Path
		}
		olderFile := fmt.Sprintf("%s.%d", l.Path, i)
		// overwrite the olderFile with the newerFile
		err := os.Rename(newerFile, olderFile)
		if os.IsNotExist(err) {
			// the newerFile does not exist
			continue
		}
		if err != nil {
			return err
		}
	}
	f, err := os.Create(l.Path)
	if err != nil {
		return err
	}
	l.File = f
	l.BytesWritten = 0
	return nil
}

func main() {
	socketPath := flag.String("socket", "/var/run/memlogdq.sock", "memlogd log query socket")
	logDir := flag.String("log-dir", "/var/log", "Directory containing log files")
	maxLogFiles := flag.Int("max-log-files", 10, "Maximum number of rotated log files before deletion")
	maxLogSize := flag.Int("max-log-size", mb, "Maximum size of a log file before rotation")
	flag.Parse()

	addr := net.UnixAddr{
		Name: *socketPath,
		Net:  "unix",
	}
	conn, err := net.DialUnix("unix", nil, &addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	n, err := conn.Write([]byte{logDumpFollow})
	if err != nil || n < 1 {
		log.Fatalf("Failed to write request to memlogd socket: %v", err)
	}

	// map of service name to active log file
	logs := make(map[string]*LogFile)

	var msg LogMessage
	decoder := json.NewDecoder(conn)
	for {
		if err := decoder.Decode(&msg); err != nil {
			log.Println(err)
			continue
		}
		if strings.HasPrefix(msg.Source, "logwrite") {
			// don't log our own output in a loop
			continue
		}

		var logF *LogFile
		var ok bool
		if logF, ok = logs[msg.Source]; !ok {
			logF, err = NewLogFile(*logDir, msg.Source)
			if err != nil {
				log.Printf("Failed to create log file %s: %v", msg.Source, err)
				continue
			}
			logs[msg.Source] = logF
		}
		if err = logF.Write(&msg); err != nil {
			log.Printf("Failed to write to log file %s: %v", msg.Source, err)
			if err := logF.Close(); err != nil {
				log.Printf("Failed to close log file %s: %v", msg.Source, err)
			}
			delete(logs, msg.Source)
			continue
		}
		if logF.BytesWritten > *maxLogSize {
			logF.Rotate(*maxLogFiles)
		}
	}
}
