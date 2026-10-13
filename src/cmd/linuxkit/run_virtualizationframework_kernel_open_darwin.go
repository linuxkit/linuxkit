//go:build darwin

package main

import (
	"os"
	"syscall"
)

func openVirtualizationFrameworkKernelPath(path string) (*os.File, error) {
	// Nonblocking open avoids waiting on FIFOs; fstat validates the opened target, including symlinks.
	return os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
}
