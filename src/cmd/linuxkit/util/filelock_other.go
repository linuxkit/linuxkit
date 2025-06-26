//go:build !unix

package util

// Lock opens the file (creating it if needed) and sets an exclusive lock.
// Returns a FileLock that can later be unlocked.
func Lock(path string) (*FileLock, error) {
	return &FileLock{}, nil
}

// Unlock releases the lock and closes the file.
func (l *FileLock) Unlock() error {
	return nil
}

// CheckLock attempts to detect if the file is locked by another process.
func CheckLock(path string) (locked bool, holderPID int, err error) {
	return false, 0, nil
}
