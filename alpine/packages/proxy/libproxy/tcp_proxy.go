package libproxy

import (
	"io"
	"net"

	"github.com/Sirupsen/logrus"
)

type Conn interface {
	io.Reader
	io.Writer
	io.Closer
	CloseRead() error
	CloseWrite() error
}

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

func HandleTCPConnection(client Conn, backendAddr *net.TCPAddr, quit chan bool) {
	backend, err := net.DialTCP("tcp", nil, backendAddr)
	if err != nil {
		logrus.Printf("Can't forward traffic to backend tcp/%v: %s\n", backendAddr, err)
		client.Close()
		return
	}

	event := make(chan int64)
	var broker = func(to, from Conn) {
		written, err := io.Copy(to, from)
		if err != nil {
			logrus.Println("error copying:", err)
		}
		err = from.CloseRead()
		if err != nil {
			logrus.Println("error CloseRead from:", err)
		}
		err = to.CloseWrite()
		if err != nil {
			logrus.Println("error CloseWrite to:", err)
		}
		event <- written
	}

	go broker(client, backend)
	go broker(backend, client)

	var transferred int64
	for i := 0; i < 2; i++ {
		select {
		case written := <-event:
			transferred += written
		case <-quit:
			// Interrupt the two brokers and "join" them.
			client.Close()
			backend.Close()
			for ; i < 2; i++ {
				transferred += <-event
			}
			return
		}
	}
	client.Close()
	backend.Close()
}

// Run starts forwarding the traffic using TCP.
func (proxy *TCPProxy) Run() {
	quit := make(chan bool)
	defer close(quit)
	for {
		client, err := proxy.listener.Accept()
		if err != nil {
			logrus.Printf("Stopping proxy on tcp/%v for tcp/%v (%s)", proxy.frontendAddr, proxy.backendAddr, err)
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
