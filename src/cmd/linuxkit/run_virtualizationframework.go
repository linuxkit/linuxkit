//go:build !darwin
// +build !darwin

package main

import (
	log "github.com/sirupsen/logrus"
)

// Process the run arguments and execute run
func runVirtualizationFramework(args []string) {
	log.Fatal("virtualization framework is available only on macOS")
}
