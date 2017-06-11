// +build !windows

package main

// Fallback implementation

import (
	"log"
)

func hypervStartConsole(vmName string) error {
	log.Fatalf("This function should not be called")
	return nil
}

func hypervRestoreConsole() {
}
