package main

import (
	"net"
	"regexp"
	"sync"
	"time"

	"github.com/digineo/go-dhclient"
	"github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
)

// DHCPServerDirectLocator This is an extreme method of trying to find a DHCP server by sending DHCP packets directly
type DHCPServerDirectLocator struct {
}

// ethernetInterfaceNamesRegex a simple regex to determine ethernet interfaces
// example: eth0, eth1, enps1p4
var ethernetInterfaceNamesRegex = regexp.MustCompile(`eth\d+|enp\d+s\d+`)

// Probe find DHCP servers
func (d *DHCPServerDirectLocator) Probe() (possibleAddresses []string, err error) {
	defer func() {
		if err == nil {
			possibleAddresses = funk.UniqString(possibleAddresses)
		}
	}()

	var ifaces []net.Interface
	ifaces, err = net.Interfaces()
	if err != nil {
		return
	}
	ifaces = funk.Filter(ifaces, func(p net.Interface) bool {
		return ethernetInterfaceNamesRegex.MatchString(p.Name)
	}).([]net.Interface)

	var wg sync.WaitGroup
	for _, iface := range ifaces {
		wg.Add(1)
		go func(wg *sync.WaitGroup, iface *net.Interface) {
			logrus.Debugf("starting DHCP broadcast on interface %s...", iface.Name)

			var client dhclient.Client
			client = dhclient.Client{Iface: iface, OnBound: func(lease *dhclient.Lease) {
				// I think this could potentially panic here because what if the wait group instead timing out?
				// then the method exits and GC kicks in causing the possibleAddresses array to be in undefined state/garbage value
				// i.e. UAF, but anyway this is just a small utility and panicking is acceptable,
				// and using select to handle these situations by gracefully stopping all spawned goroutines would probably be nicer
				possibleAddresses = append(possibleAddresses, lease.ServerID.String())
				go client.Stop()
				wg.Done()
			}}
			client.Start()
		}(&wg, &iface)
	}

	// either wait group finishes or time out after 30 seconds
	waitTimeout(&wg, 30*time.Second)
	return
}
