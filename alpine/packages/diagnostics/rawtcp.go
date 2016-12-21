package main

import (
	"log"
	"net"
)

// RawTCPDiagnosticListener is a diagnostic server listening on a TCP port
type RawTCPDiagnosticListener struct{}

// Listen for RawTCPDiagnosticListener listens on port 62374
func (l RawTCPDiagnosticListener) Listen() {
	ip, err := net.Listen("tcp", ":62374")
	if err != nil {
		log.Printf("Failed to bind to TCP port 62374: %s", err)
	}

	for {
		TarRespond(ip)
	}
}
