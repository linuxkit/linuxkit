package libproxy

import (
	"fmt"
	"net"
)

// TCPProxy is a proxy for TCP connections. It implements the Proxy interface to
// handle TCP traffic forwarding between the frontend and backend addresses.
type TCPProxy struct {
	listener     net.Listener
	frontendAddr net.Addr
	backendAddr  *net.TCPAddr
}

// NewTCPProxy creates a new TCPProxy.
func NewTCPProxy(listener net.Listener, backendAddr *net.TCPAddr) (*TCPProxy, error) {
	// If the port in frontendAddr was 0 then ListenTCP will have a picked
	// a port to listen on, hence the call to Addr to get that actual port:
	return &TCPProxy{
		listener:     listener,
		frontendAddr: listener.Addr(),
		backendAddr:  backendAddr,
	}, nil
}

// HandleTCPConnection forwards the TCP traffic to a specified backend address
func HandleTCPConnection(client Conn, backendAddr *net.TCPAddr, quit <-chan struct{}) error {
	backend, err := net.DialTCP("tcp", nil, backendAddr)
	if err != nil {
		if errIsConnectionRefused(err) {
			return err
		}
		return fmt.Errorf("can't forward traffic to backend tcp/%v: %s", backendAddr, err)
	}
	return ProxyStream(client, backend, quit)
}

// Run starts forwarding the traffic using TCP.
func (proxy *TCPProxy) Run() {
	quit := make(chan struct{})
	defer close(quit)
	for {
		client, err := proxy.listener.Accept()
		if err != nil {
			log.Printf("Stopping proxy on tcp/%v for tcp/%v (%s)", proxy.frontendAddr, proxy.backendAddr, err)
			return
		}
		go HandleTCPConnection(client.(Conn), proxy.backendAddr, quit)
	}
}

// Close stops forwarding the traffic.
func (proxy *TCPProxy) Close() { proxy.listener.Close() }

// FrontendAddr returns the TCP address on which the proxy is listening.
func (proxy *TCPProxy) FrontendAddr() net.Addr { return proxy.frontendAddr }

// BackendAddr returns the TCP proxied address.
func (proxy *TCPProxy) BackendAddr() net.Addr { return proxy.backendAddr }
