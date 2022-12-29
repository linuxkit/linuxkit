//go:build darwin && !cgo
// +build darwin,!cgo

package main

import (
	"errors"
)

// Process the run arguments and execute run
func runVirtualizationFramework(cfg virtualizationFramwworkConfig, image string) error {
	return errors.New("This build of linuxkit was compiled without virtualization framework capabilities. " +
		"To perform 'linuxkit run' on macOS, please use a version of linuxkit compiled with virtualization framework.")
}
