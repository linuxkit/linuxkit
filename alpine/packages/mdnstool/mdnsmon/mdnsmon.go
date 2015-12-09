package mdnsmon

// mDNS server that publishes a service with the IP address(es) of a monitored network interface.

import (
	"log"
	"net"
	"time"

	"github.com/hashicorp/mdns"
)

type MDNSServer struct {
	service    *mdns.MDNSService
	iface      *net.Interface
	ip_updates chan []net.IP
	shutdown   chan int
}

// NewServer creates a new mDNS service and server configuration.
func NewServer(hostname string, instance string, port int, srv string, info []string, iface *net.Interface) (*MDNSServer, error) {
	service, err := mdns.NewMDNSService(instance, srv, "local.", hostname, port, []net.IP{net.ParseIP("0.0.0.0")}, info)
	if err != nil {
		return nil, err
	}

	ip_updates := make(chan []net.IP)
	shutdown := make(chan int)

	return &MDNSServer{service: service, ip_updates: ip_updates, shutdown: shutdown, iface: iface}, nil
}

// getIPs gets a list of IP addresses from an interface
func getIPs(iface *net.Interface) []net.IP {
	addrs, err := iface.Addrs()
	if err != nil {
		log.Printf("Unable to read interface address(es), error: %s", err)
		return []net.IP{}
	}

	var ips []net.IP
	for _, a := range addrs {
		switch v := a.(type) {
		case *net.IPNet:
			if v.IP.To4() != nil { // Only support IPv4 for now
				ips = append(ips, v.IP)
			}
		}
	}

	return ips
}

func (m *MDNSServer) runServer() {
	var server *mdns.Server
	var err error

	defer func() {
		if server != nil {
			log.Println("Shutting down mDNS server...")
			server.Shutdown()
		}
	}()

	for {
		select {
		case ips := <-m.ip_updates: // New IP set received, registering service
			// Update service/zone record
			m.service.IPs = ips

			// Shutdown old mDNS server, if running
			if server != nil {
				log.Println("New configuration - shutting down mDNS server...")
				server.Shutdown()
				time.Sleep(1 * time.Second)
				server = nil
			}

			// Skip if no IPs found
			if len(ips) == 0 {
				log.Println("No IP address. Waiting for IP to be configured.")
				continue
			}

			// Create the mDNS server
			log.Println("Answering requests for IP(s) ", ips)
			server, err = mdns.NewServer(&mdns.Config{Zone: m.service, Iface: m.iface})
			if err != nil {
				log.Println(err)
				m.service.IPs = []net.IP{} // Reset IP set so we can automatically retry later
			}
		case <-m.shutdown:
			break
		}
	}
}

// isIPsequal compares to slices with IPs
func isIPsequal(a []net.IP, b []net.IP) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil && b != nil {
		return false
	}
	if a != nil && b == nil {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for _, ip1 := range a {
		match := false
		for _, ip2 := range b {
			if ip1.Equal(ip2) {
				match = true
				break
			}
		}
		if match == false { // if one ip from a is not in b, return false
			return false
		}
	}
	return true
}

// Start starts the background mDNS server and starts monitoring the network interface for IP changes.
func (m *MDNSServer) Start() {
	// Start background server
	go m.runServer()

	// Monitor interface for IP changes
	for {
		ips := getIPs(m.iface)
		if !isIPsequal(ips, m.service.IPs) {
			log.Println("IP configuration:", ips)
			m.ip_updates <- ips
		}

		//TODO(magnuss) Monitor using netlink?
		if len(ips) == 0 { // Sleep shorter if no IP found
			time.Sleep(1 * time.Second)
		} else {
			time.Sleep(60 * time.Second) // Takes longer to react on IP change, but mDNS has TTL of 120 sec
		}
	}

}

// Shutdown stops the background mDNS server and stops monitorint the network interface for IP changes.
func (m *MDNSServer) Shutdown() {
	m.shutdown <- 0 // TODO(magnuss) Wait for mDNS to shutdown
}
