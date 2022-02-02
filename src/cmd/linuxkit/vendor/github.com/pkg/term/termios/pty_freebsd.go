package termios

import (
	"fmt"
	"unsafe"
)

func posix_openpt(oflag int) (fd uintptr, err error) {
	// Copied from debian-golang-pty/pty_freebsd.go.
	r0, _, e1 := unix.Syscall(unix.SYS_POSIX_OPENPT, uintptr(oflag), 0, 0)
	fd = uintptr(r0)
	if e1 != 0 {
		err = e1
	}
	return
}

func open_pty_master() (uintptr, error) {
	return posix_openpt(unix.O_NOCTTY | unix.O_RDWR | unix.O_CLOEXEC)
}

func Ptsname(fd uintptr) (string, error) {
	var n uintptr
	err := ioctl(fd, unix.TIOCGPTN, uintptr(unsafe.Pointer(&n)))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("/dev/pts/%d", n), nil
}

func grantpt(fd uintptr) error {
	var n uintptr
	return ioctl(fd, unix.TIOCGPTN, uintptr(unsafe.Pointer(&n)))
}

func unlockpt(fd uintptr) error {
	return nil
}
