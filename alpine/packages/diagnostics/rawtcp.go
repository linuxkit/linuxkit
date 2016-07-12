package main

import (
	"log"
	"net"
)

type RawTCPDiagnosticListener struct{}

func (l RawTCPDiagnosticListener) Listen() {
	ip, err := net.Listen("tcp", ":62374")
	if err != nil {
		log.Printf("Failed to bind to TCP port 62374: %s", err)
	}

	for {
		TarRespond(ip)
	}
}
