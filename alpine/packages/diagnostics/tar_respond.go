package main

import (
	"archive/tar"
	"log"
	"net"
)

// TarRespond is used to write back a tar archive over a connection for the
// lower-level listener types.
//
// In local virtualization (which this is for) we write back a tar file
// directly and the client takes care of shipping the result to the mothership.
//
// By contrast, in cloud editions we expect each node to ship the captured
// information on its own, so this function is not used.
//
// This is a deliberate design to choice to ensure that it is possible in the
// future for diagnostic information to be reported from nodes which have have
// been separated via network partition from the node which initiates
// diagnostic collection, and/or if we decide to automatically collect
// diagnostic information from nodes which deem *themselves* unhealthy at a
// future time.
func TarRespond(l net.Listener) {
	conn, err := l.Accept()
	if err != nil {
		log.Printf("Error accepting connection: %s", err)
		return
	}

	w := tar.NewWriter(conn)

	Capture(w, localCaptures)

	if err := w.Close(); err != nil {
		log.Println(err)
	}

	conn.Close()
}
