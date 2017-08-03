package main

// Implement Windows specific functions here
import (
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/Azure/go-ansiterm/winterm"
	"github.com/Microsoft/go-winio"
	log "github.com/sirupsen/logrus"
)

// Some of the code below is copied and modified from:
// https://github.com/moby/moby/blob/master/pkg/term/term_windows.go
const (
	// https://msdn.microsoft.com/en-us/library/windows/desktop/ms683167(v=vs.85).aspx
	enableVirtualTerminalInput      = 0x0200
	enableVirtualTerminalProcessing = 0x0004
	disableNewlineAutoReturn        = 0x0008
)

func hypervStartConsole(vmName string) error {
	if err := hypervConfigureConsole(); err != nil {
		log.Infof("Configure Console: %v", err)
	}

	pipeName := fmt.Sprintf(`\\.\pipe\%s-com1`, vmName)
	var c net.Conn
	var err error
	for count := 1; count < 100; count++ {
		c, err = winio.DialPipe(pipeName, nil)
		defer c.Close()
		if err != nil {
			// Argh, different Windows versions seem to
			// return different errors and we can't easily
			// catch the error. On some versions it is
			// winio.ErrTimeout...
			// Instead poll 100 times and then error out
			log.Infof("Connect to console: %v", err)
			time.Sleep(10 * 1000 * 1000 * time.Nanosecond)
			continue
		}
		break
	}
	if err != nil {
		return err
	}

	log.Info("Connected")
	go io.Copy(c, os.Stdin)

	_, err = io.Copy(os.Stdout, c)
	if err != nil {
		return err
	}
	return nil
}

var (
	hypervStdinMode  uint32
	hypervStdoutMode uint32
	hypervStderrMode uint32
)

func hypervConfigureConsole() error {
	// Turn on VT handling on all std handles, if possible. This might
	// fail on older windows version, but we'll ignore that for now
	// Also disable local echo

	fd := os.Stdin.Fd()
	if hypervStdinMode, err := winterm.GetConsoleMode(fd); err == nil {
		if err = winterm.SetConsoleMode(fd, hypervStdinMode|enableVirtualTerminalInput); err != nil {
			log.Warn("VT Processing is not supported on stdin")

		}
	}

	fd = os.Stdout.Fd()
	if hypervStdoutMode, err := winterm.GetConsoleMode(fd); err == nil {
		if err = winterm.SetConsoleMode(fd, hypervStdoutMode|enableVirtualTerminalProcessing|disableNewlineAutoReturn); err != nil {
			log.Warn("VT Processing is not supported on stdout")
		}
	}

	fd = os.Stderr.Fd()
	if hypervStderrMode, err := winterm.GetConsoleMode(fd); err == nil {
		if err = winterm.SetConsoleMode(fd, hypervStderrMode|enableVirtualTerminalProcessing|disableNewlineAutoReturn); err != nil {
			log.Warn("VT Processing is not supported on stderr")
		}
	}
	return nil
}

func hypervRestoreConsole() {
	winterm.SetConsoleMode(os.Stdin.Fd(), hypervStdinMode)
	winterm.SetConsoleMode(os.Stdout.Fd(), hypervStdoutMode)
	winterm.SetConsoleMode(os.Stderr.Fd(), hypervStderrMode)
}
