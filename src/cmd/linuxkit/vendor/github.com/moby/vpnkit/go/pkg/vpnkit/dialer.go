package vpnkit

import (
	"context"
	"errors"
	"net"
	"strconv"
	"sync"

	"github.com/moby/vpnkit/go/pkg/libproxy"
)

// DefaultVsockPort is the default AF_VSOCK port where vpnkit-forwarder listens.
const DefaultVsockPort = 62373

// Dialer connects to remote addresses via the vpnkit-forwarder.
type Dialer struct {
	HyperkitConnectPath string // HyperkitConnectPath is the path of the `connect` Unix domain socket
	HyperVVMID          string // HyperkitVMVMID is the GUID of the VM running vpnkit-forwarder
	Port                int    // Port is the AF_VSOCK port where vpnkit-forwarder is listening
	m                   sync.Mutex
	mux                 libproxy.Multiplexer
}

func (d *Dialer) setupMultiplexer() error {
	d.m.Lock()
	defer d.m.Unlock()
	if d.mux != nil && !d.mux.IsRunning() {
		d.mux.Close()
		d.mux = nil
	}
	if d.mux == nil {
		conn, err := d.connectTransport()
		if err != nil {
			return err
		}
		d.mux, err = libproxy.NewMultiplexer("host", conn, true)
		if err != nil {
			return err
		}
		d.mux.Run()
	}
	return nil
}

// Dial connects to the address on the named network.
func (d *Dialer) Dial(network, address string) (net.Conn, error) {
	if err := d.setupMultiplexer(); err != nil {
		return nil, err
	}
	switch network {
	case "unix":
		return d.mux.Dial(libproxy.Destination{
			Proto: libproxy.Unix,
			Path:  address,
		})
	case "udp", "tcp":
		host, portS, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		port, err := strconv.Atoi(portS)
		if err != nil {
			return nil, err
		}
		resolver := &net.Resolver{}
		addrs, err := resolver.LookupIPAddr(context.Background(), host)
		if err != nil {
			return nil, err
		}
		if len(addrs) == 0 {
			return nil, errors.New("Failed to resolve IP of " + host)
		}
		var ip net.IP
		// we only support IPv4 for now, so pick the first
		for _, addr := range addrs {
			if ipv4 := addr.IP.To4(); ipv4 != nil && ip == nil {
				ip = addr.IP
				break
			}
		}
		if ip == nil {
			return nil, errors.New("Failed to resolve an IPv4 of " + host)
		}
		proto := libproxy.UDP
		if network == "tcp" {
			proto = libproxy.TCP
		}
		return d.mux.Dial(libproxy.Destination{
			Proto: proto,
			IP:    ip,
			Port:  uint16(port),
		})
	default:
		return nil, errors.New("unknown network: " + network)
	}
}
