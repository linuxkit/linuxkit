package libproxy

import (
	"fmt"
	"net"
)

// Forward a connection to a given destination.
func Forward(conn Conn, destination Destination, quit <-chan struct{}) {
	defer conn.Close()

	switch destination.Proto {
	case TCP:
		backendAddr := net.TCPAddr{IP: destination.IP, Port: int(destination.Port), Zone: ""}
		if err := HandleTCPConnection(conn, &backendAddr, quit); err != nil {
			log.Printf("closing TCP proxy because %v", err)
			return
		}
	case Unix:
		backendAddr, err := net.ResolveUnixAddr("unix", destination.Path)
		if err != nil {
			log.Printf("Error resolving Unix address %s", destination.Path)
			return
		}
		if err := HandleUnixConnection(conn, backendAddr, quit); err != nil {
			log.Printf("closing Unix proxy because %v", err)
			return
		}
	case UDP:
		backendAddr := &net.UDPAddr{IP: destination.IP, Port: int(destination.Port), Zone: ""}
		// copy to and from the backend without using NewUDPProxy
		inside, err := net.DialUDP("udp", nil, backendAddr)
		if err != nil {
			log.Printf("Failed to Dial UDP backend for %s: %v", backendAddr, err)
			return
		}
		log.Printf("accepted UDP connection to %s\n", backendAddr.String())
		one := make(chan struct{})
		two := make(chan struct{})
		go func() {
			copyUDP(fmt.Sprintf("from %s to host", backendAddr.String()), inside, conn)
			close(one)
		}()
		go func() {
			copyUDP(fmt.Sprintf("from host to %s", backendAddr.String()), conn, inside)
			close(two)
		}()
		select {
		case <-quit: // we want to quit
		case <-one: // we get an error like "connection refused"
		case <-two: // we get an error like "connection refused"
		}
		log.Printf("closing UDP connection to %s\n", backendAddr.String())
		_ = inside.Close()
		return
	default:
		log.Printf("Unknown protocol: %d", destination.Proto)
		return
	}
}

func copyUDP(description string, left, right net.Conn) {
	b := make([]byte, UDPBufSize)
	for {
		n, err := left.Read(b)
		if err != nil {
			log.Printf("%s: unable to read UDP: %v", description, err)
			return
		}
		pkt := b[0:n]
		_, err = right.Write(pkt)
		if err != nil {
			log.Printf("%s: unable to write UDP: %v", description, err)
			return
		}
	}
}
