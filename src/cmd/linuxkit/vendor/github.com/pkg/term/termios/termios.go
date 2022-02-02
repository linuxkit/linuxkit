// +build !windows

package termios

import (
	"golang.org/x/sys/unix"
)

// Tiocmget returns the state of the MODEM bits.
func Tiocmget(fd uintptr) (int, error) {
	return unix.IoctlGetInt(int(fd), unix.TIOCMGET)
}

// Tiocmset sets the state of the MODEM bits.
func Tiocmset(fd uintptr, status int) error {
	return unix.IoctlSetInt(int(fd), unix.TIOCMSET, status)
}

// Tiocmbis sets the indicated modem bits.
func Tiocmbis(fd uintptr, status int) error {
	return unix.IoctlSetPointerInt(int(fd), unix.TIOCMBIS, status)
}

// Tiocmbic clears the indicated modem bits.
func Tiocmbic(fd uintptr, status int) error {
	return unix.IoctlSetPointerInt(int(fd), unix.TIOCMBIC, status)
}

// Cfmakecbreak modifies attr for cbreak mode.
func Cfmakecbreak(attr *unix.Termios) {
	attr.Lflag &^= unix.ECHO | unix.ICANON
	attr.Cc[unix.VMIN] = 1
	attr.Cc[unix.VTIME] = 0
}

// Cfmakeraw modifies attr for raw mode.
func Cfmakeraw(attr *unix.Termios) {
	attr.Iflag &^= unix.BRKINT | unix.ICRNL | unix.INPCK | unix.ISTRIP | unix.IXON
	attr.Oflag &^= unix.OPOST
	attr.Cflag &^= unix.CSIZE | unix.PARENB
	attr.Cflag |= unix.CS8
	attr.Lflag &^= unix.ECHO | unix.ICANON | unix.IEXTEN | unix.ISIG
	attr.Cc[unix.VMIN] = 1
	attr.Cc[unix.VTIME] = 0
}
