//go:build !darwin

package main

import (
	"fmt"
	"os"
)

func openVirtualizationFrameworkKernelPath(path string) (*os.File, error) {
	// Portable fallback for builds without the Darwin nonblocking opener.
	// Reject special files before os.Open; the caller also validates the opened file.
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("unable to stat kernel path: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("kernel path is not a regular file (mode %s)", info.Mode())
	}

	return os.Open(path)
}
