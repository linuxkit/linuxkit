package main

import (
	"github.com/sirupsen/logrus"
)

// DHCPServerLocator locate DHCP servers
type DHCPServerLocator interface {
	Probe() (possibleServers []string, err error)
}

// FindPossibleDHCPServers find possible dhcp servers
func FindPossibleDHCPServers() (possibleDhcpServAddr []string, err error) {
	var locatorFacade interface{}

	state := "file"
loop:
	for {
		switch state {
		case "file":
			logrus.WithField("method", "existing DHCP lease information").
				Infof("checking DHCP servers")
			locatorFacade = &DHCPServerLeaseFileLocator{}
			state = "run"
		case "direct":
			logrus.WithField("method", "directly send DHCP packets").
				Infof("checking DHCP servers")
			locatorFacade = &DHCPServerDirectLocator{}
			state = "run"
		case "run":
			if locator, ok := locatorFacade.(DHCPServerLocator); ok {
				possibleDhcpServAddr, err = locator.Probe()
				if err != nil {
					if _, ok := locatorFacade.(*DHCPServerLeaseFileLocator); ok {
						logrus.WithError(err).
							Warn("cannot find DHCP server")
						state = "direct"
					} else {
						state = "fail"
					}
				} else {
					state = "Success"
				}
			} else {
				state = "fail"
			}
		case "fail":
			logrus.
				Debug("unable to find DHCP servers with known methods")
			break loop
		case "Success":
			logrus.WithField("servers", possibleDhcpServAddr).
				Debugf("found DHCP servers")
			break loop
		}
	}
	return
}
