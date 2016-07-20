package main

import (
	"log"
	"syscall"

	"github.com/rneugeba/virtsock/go/vsock"
)

type VSockDiagnosticListener struct{}

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
