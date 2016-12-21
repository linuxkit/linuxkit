package main

import (
	"log"
	"syscall"

	"github.com/rneugeba/virtsock/go/hvsock"
)

// HVSockDiagnosticListener is a diagnostic server using HVSock (Windows)
type HVSockDiagnosticListener struct{}

// Listen sets up the diagnostic server to listen on an HVSock
func (l HVSockDiagnosticListener) Listen() {
	svcid, _ := hvsock.GuidFromString("445BA2CB-E69B-4912-8B42-D7F494D007EA")
	hvsock, err := hvsock.Listen(hvsock.HypervAddr{VmId: hvsock.GUID_WILDCARD, ServiceId: svcid})
	if err != nil {
		if errno, ok := err.(syscall.Errno); !ok || errno != syscall.EAFNOSUPPORT {
			log.Printf("Failed to bind to hvsock port: %s", err)
		}
	}

	for {
		TarRespond(hvsock)
	}
}
