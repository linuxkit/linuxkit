//go:build !darwin

package main

import (
	"errors"
	"fmt"
	"os"
)

func openVirtualizationFrameworkKernelFile(path string) (*os.File, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("unable to stat kernel path: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("kernel path is not a regular file (mode %s)", info.Mode())
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open kernel path: %w", err)
	}
	info, err = file.Stat()
	if err != nil {
		return nil, errors.Join(fmt.Errorf("unable to stat opened kernel path: %w", err), file.Close())
	}
	if !info.Mode().IsRegular() {
		return nil, errors.Join(fmt.Errorf("kernel path is not a regular file (mode %s)", info.Mode()), file.Close())
	}

	return file, nil
}
