//go:build darwin && !cgo
// +build darwin,!cgo

package main

import (
	log "github.com/sirupsen/logrus"
)

// Process the run arguments and execute run
func runVirtualizationFramework(args []string) {
	log.Fatal("This build of linuxkit was compiled without virtualization framework capabilities. " +
		"To perform 'linuxkit run' on macOS, please use a version of linuxkit compiled with virtualization framework.")
}
