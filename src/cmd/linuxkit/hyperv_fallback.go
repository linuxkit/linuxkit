//go:build !windows
// +build !windows

package main

// Fallback implementation

import (
	"log"
)

//nolint:unused
func hypervStartConsole(vmName string) error {
	log.Fatalf("This function should not be called")
	return nil
}

//nolint:unused
func hypervRestoreConsole() {
}
