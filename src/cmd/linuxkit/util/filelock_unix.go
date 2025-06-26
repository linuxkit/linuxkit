//go:build unix

package util

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/sys/unix"
)

// Lock opens the file (creating it if needed) and sets an exclusive lock.
// Returns a FileLock that can later be unlocked.
func Lock(path string) (*FileLock, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}

	flock := unix.Flock_t{
		Type:   unix.F_WRLCK,
		Whence: int16(io.SeekStart),
		Start:  0,
		Len:    0,
	}

	if err := unix.FcntlFlock(f.Fd(), unix.F_SETLKW, &flock); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("set lock: %w", err)
	}

	return &FileLock{file: f}, nil
}

// Unlock releases the lock and closes the file.
func (l *FileLock) Unlock() error {
	flock := unix.Flock_t{
		Type:   unix.F_UNLCK,
		Whence: int16(io.SeekStart),
		Start:  0,
		Len:    0,
	}
	if err := unix.FcntlFlock(l.file.Fd(), unix.F_SETLKW, &flock); err != nil {
		return fmt.Errorf("unlock: %w", err)
	}
	return l.file.Close()
}

// CheckLock attempts to detect if the file is locked by another process.
func CheckLock(path string) (locked bool, holderPID int, err error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return false, 0, fmt.Errorf("open file: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	check := unix.Flock_t{
		Type:   unix.F_WRLCK,
		Whence: int16(io.SeekStart),
		Start:  0,
		Len:    0,
	}

	if err := unix.FcntlFlock(f.Fd(), unix.F_GETLK, &check); err != nil {
		return false, 0, fmt.Errorf("get lock: %w", err)
	}

	if check.Type == unix.F_UNLCK {
		return false, 0, nil
	}
	return true, int(check.Pid), nil
}

// WaitUnlocked waits until the file is unlocked by another process, and uses it for reading but not writing.
func WaitUnlocked(path string) error {
	f, err := os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	flock := unix.Flock_t{
		Type:   unix.F_RDLCK,
		Whence: int16(io.SeekStart),
		Start:  0,
		Len:    0,
	}

	if err := unix.FcntlFlock(f.Fd(), unix.F_SETLKW, &flock); err != nil {
		_ = f.Close()
		return fmt.Errorf("set lock: %w", err)
	}
	fileRef := &FileLock{file: f}
	_ = fileRef.Unlock()
	return nil
}
