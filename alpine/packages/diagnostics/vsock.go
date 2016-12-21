package main

import (
	"log"
	"syscall"

	"github.com/rneugeba/virtsock/go/vsock"
)

// VSockDiagnosticListener is a diagnostic server listening on VSock
type VSockDiagnosticListener struct{}

// Listen for VSockDiagnosticListener listens on a VSock's port 62374
func (l VSockDiagnosticListener) Listen() {
	vsock, err := vsock.Listen(uint(62374))
	if err != nil {
		if errno, ok := err.(syscall.Errno); !ok || errno != syscall.EAFNOSUPPORT {
			log.Printf("Failed to bind to vsock port 62374: %s", err)
		}
	}

	for {
		TarRespond(vsock)
	}
}
