package main

import (
	"errors"
	"fmt"
	"os"
)

func openVirtualizationFrameworkKernelFile(path string) (*os.File, error) {
	file, err := openVirtualizationFrameworkKernelPath(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open kernel path: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		return nil, errors.Join(fmt.Errorf("unable to stat opened kernel path: %w", err), file.Close())
	}
	if !info.Mode().IsRegular() {
		return nil, errors.Join(fmt.Errorf("kernel path is not a regular file (mode %s)", info.Mode()), file.Close())
	}
	return file, nil
}
