//go:build !darwin
// +build !darwin

package main

import (
	"errors"
)

// Process the run arguments and execute run
func runVirtualizationFramework(cfg virtualizationFramwworkConfig, image string) error {
	return errors.New("virtualization framework is available only on macOS")
}
