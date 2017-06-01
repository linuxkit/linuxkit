// +build !linux,!darwin

package vsock

import (
	"fmt"
	"log"
	"net"
)

// SocketMode is the unimplemented fallback for unsupported OSes
func SocketMode(socketMode string) {
	log.Fatalln("Unimplemented")
}

// Dial is the unimplemented fallback for unsupported OSes
func Dial(cid, port uint) (Conn, error) {
	return nil, fmt.Errorf("Unimplemented")
}

// Listen is the unimplemented fallback for unsupported OSes
func Listen(cid, port uint) (net.Listener, error) {
	return nil, fmt.Errorf("Unimplemented")
}
