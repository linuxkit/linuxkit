// Package libproxy provides a network Proxy interface and implementations for TCP
// and UDP.
package libproxy

import (
	"fmt"
	"net"
	"os"
	"syscall"
)

// Proxy defines the behavior of a proxy. It forwards traffic back and forth
// between two endpoints : the frontend and the backend.
// It can be used to do software port-mapping between two addresses.
// e.g. forward all traffic between the frontend (host) 127.0.0.1:3000
// to the backend (container) at 172.17.42.108:4000.
type Proxy interface {
	// Run starts forwarding traffic back and forth between the front
	// and back-end addresses.
	Run()
	// Close stops forwarding traffic and close both ends of the Proxy.
	Close()
	// FrontendAddr returns the address on which the proxy is listening.
	FrontendAddr() net.Addr
	// BackendAddr returns the proxied address.
	BackendAddr() net.Addr
}

// NewIPProxy creates a Proxy according to the specified frontendAddr and backendAddr.
func NewIPProxy(frontendAddr, backendAddr net.Addr) (Proxy, error) {
	switch frontendAddr.(type) {
	case *net.UDPAddr:
		listener, err := net.ListenUDP("udp", frontendAddr.(*net.UDPAddr))
		if err != nil {
			return nil, err
		}
		return NewUDPProxy(listener.LocalAddr().(*net.UDPAddr), listener, backendAddr.(*net.UDPAddr), nil)
	case *net.TCPAddr:
		listener, err := net.Listen("tcp", frontendAddr.String())
		if err != nil {
			return nil, err
		}
		return NewTCPProxy(listener, backendAddr.(*net.TCPAddr))
	case *net.UnixAddr:
		listener, err := net.Listen("unix", frontendAddr.String())
		if err != nil {
			return nil, err
		}
		return NewUnixProxy(listener, backendAddr.(*net.UnixAddr))
	default:
		panic(fmt.Errorf("Unsupported protocol"))
	}
}

// NewBestEffortIPProxy Best-effort attempt to listen on the address in the VM. This is for
// backwards compatibility with software that expects to be able to listen on
// 0.0.0.0 and then connect from within a container to the external port.
// If the address doesn't exist in the VM (i.e. it exists only on the host)
// then this is not a hard failure.
func NewBestEffortIPProxy(host net.Addr, container net.Addr) (Proxy, error) {
	ipP, err := NewIPProxy(host, container)
	if err == nil {
		return ipP, nil
	}
	if opError, ok := err.(*net.OpError); ok {
		if syscallError, ok := opError.Err.(*os.SyscallError); ok {
			if syscallError.Err == syscall.EADDRNOTAVAIL {
				log.Printf("Address %s doesn't exist in the VM: only binding on the host", host)
				return nil, nil // Non-fatal error
			}
		}
	}
	return nil, err
}
