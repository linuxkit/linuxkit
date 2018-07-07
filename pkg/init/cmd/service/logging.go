package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
)

var (
	errLoggingNotEnabled = errors.New("logging system not enabled")
	logWriteSocket       = "/var/run/linuxkit-external-logging.sock"
	logReadSocket        = "/var/run/memlogdq.sock"
)

const (
	logDumpCommand byte = iota
)

// Log provides access to a log by path or io.WriteCloser
type Log interface {
	Path(string) string                  // Path of the log file (may be a FIFO)
	Open(string) (io.WriteCloser, error) // Opens a log stream
	Dump(string)                         // Copies logs to the console
}

// GetLog returns the log destination we should use.
func GetLog(logDir string) Log {
	// is an external logging system enabled?
	if _, err := os.Stat(logWriteSocket); !os.IsNotExist(err) {
		return &remoteLog{
			fifoDir: "/var/run",
		}
	}
	return &fileLog{
		dir: logDir,
	}
}

type fileLog struct {
	dir string
}

func (f *fileLog) localPath(n string) string {
	return filepath.Join(f.dir, n+".log")
}

// Path returns the name of a log file path for the named service.
func (f *fileLog) Path(n string) string {
	path := f.localPath(n)
	// We just need this to exist, otherwise containerd will say:
	//
	//   ERRO[0000] failed to create task error="failed to start io pipe
	//     copy: containerd-shim: opening /var/log/... failed: open
	//     /var/log/...: no such file or directory: unknown"
	file, err := os.Create(path)
	if err != nil {
		// If we cannot write to the directory, we'll discard output instead.
		return "/dev/null"
	}
	_ = file.Close()
	return path
}

// Open a log file for the named service.
func (f *fileLog) Open(n string) (io.WriteCloser, error) {
	return os.OpenFile(f.localPath(n), os.O_WRONLY|os.O_CREATE, 0644)
}

// Dump copies logs to the console.
func (f *fileLog) Dump(n string) {
	path := f.localPath(n)
	if err := dumpFile(os.Stdout, path); err != nil {
		fmt.Printf("Error writing %s to console: %v", path, err)
	}
}

type remoteLog struct {
	fifoDir string
}

// Path returns the name of a FIFO connected to the logging daemon.
func (r *remoteLog) Path(n string) string {
	path := filepath.Join(r.fifoDir, n+".log")
	if err := syscall.Mkfifo(path, 0600); err != nil {
		return "/dev/null"
	}
	go func() {
		// In a goroutine because Open of the FIFO will block until
		// containerd opens it when the task is started.
		fd, err := syscall.Open(path, syscall.O_RDONLY, 0)
		if err != nil {
			// Should never happen: we just created the fifo
			log.Printf("failed to open fifo %s: %s", path, err)
		}
		defer syscall.Close(fd)
		if err := sendToLogger(n, fd); err != nil {
			// Should never happen: logging is enabled
			log.Printf("failed to send fifo %s to logger: %s", path, err)
		}
	}()
	return path
}

// Open a log file for the named service.
func (r *remoteLog) Open(n string) (io.WriteCloser, error) {
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		log.Fatal("Unable to create socketpair: ", err)
	}
	logFile := os.NewFile(uintptr(fds[0]), "")

	if err := sendToLogger(n, fds[1]); err != nil {
		return nil, err
	}
	return logFile, nil
}

// Dump copies logs to the console.
func (r *remoteLog) Dump(n string) {
	addr := net.UnixAddr{
		Name: logReadSocket,
		Net:  "unix",
	}
	conn, err := net.DialUnix("unix", nil, &addr)
	if err != nil {
		log.Printf("Failed to connect to logger: %s", err)
		return
	}
	defer conn.Close()
	nWritten, err := conn.Write([]byte{logDumpCommand})
	if err != nil || nWritten < 1 {
		log.Printf("Failed to request logs from logger: %s", err)
		return
	}
	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Printf("Failed to read log message: %s", err)
			return
		}
		// a line is of the form
		// <timestamp>,<log>;<body>
		prefixBody := strings.SplitN(line, ";", 2)
		csv := strings.Split(prefixBody[0], ",")
		if len(csv) < 2 {
			log.Printf("Failed to parse log message: %s", line)
			continue
		}
		if csv[1] == n {
			fmt.Print(line)
		}
	}
}

func sendToLogger(name string, fd int) error {
	var ctlSocket int
	var err error
	if ctlSocket, err = syscall.Socket(syscall.AF_UNIX, syscall.SOCK_DGRAM, 0); err != nil {
		return err
	}

	var ctlConn net.Conn
	if ctlConn, err = net.FileConn(os.NewFile(uintptr(ctlSocket), "")); err != nil {
		return err
	}
	defer ctlConn.Close()

	ctlUnixConn, ok := ctlConn.(*net.UnixConn)
	if !ok {
		// should never happen
		log.Fatal("Internal error, invalid cast.")
	}

	raddr := net.UnixAddr{Name: logWriteSocket, Net: "unixgram"}
	oobs := syscall.UnixRights(fd)
	_, _, err = ctlUnixConn.WriteMsgUnix([]byte(name), oobs, &raddr)
	if err != nil {
		return errLoggingNotEnabled
	}
	return nil
}
