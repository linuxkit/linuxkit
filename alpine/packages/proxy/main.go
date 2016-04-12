package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"pkg/proxy"
	"strings"
)

func main() {
	host, port, container := parseHostContainerAddrs()

	err := exposePort(host, port)
	if err != nil {
		sendError(err)
	}
	p, err := proxy.NewProxy(host, container)
	if err != nil {
		unexposePort(host)
		sendError(err)
	}
	go handleStopSignals(p)
	sendOK()
	p.Run()
	unexposePort(host)
	os.Exit(0)
}

func exposePort(host net.Addr, port int) error {
	name := host.String()
	log.Printf("exposePort %s\n", name)
	err := os.Mkdir("/port/"+name, 0)
	if err != nil {
		log.Printf("Failed to mkdir /port/%s: %#v\n", name, err)
		return err
	}
	ctl, err := os.OpenFile("/port/"+name+"/ctl", os.O_RDWR, 0)
	if err != nil {
		log.Printf("Failed to open /port/%s/ctl: %#v\n", name, err)
		return err
	}
	me, err := getMyAddress()
	if err != nil {
		log.Printf("Failed to determine my local address: %#v\n", err)
		return err
	}
	_, err = ctl.WriteString(fmt.Sprintf("%s:%s:%d", name, me, port))
	if err != nil {
		log.Printf("Failed to open /port/%s/ctl: %#v\n", name, err)
		return err
	}
	_, err = ctl.Seek(0, 0)
	if err != nil {
		log.Printf("Failed to seek on /port/%s/ctl: %#v\n", name, err)
		return err
	}
	results := make([]byte, 100)
	count, err := ctl.Read(results)
	if err != nil {
		log.Printf("Failed to read from /port/%s/ctl: %#v\n", name, err)
		return err
	}
	// We deliberately keep the control file open since 9P clunk
	// will trigger a shutdown on the host side.

	response := string(results[0:count])
	if strings.HasPrefix(response, "ERROR ") {
		os.Remove("/port/" + name + "/ctl")
		response = strings.Trim(response[6:], " \t\r\n")
		return errors.New(response)
	}

	return nil
}

func unexposePort(host net.Addr) {
	name := host.String()
	log.Printf("unexposePort %s\n", name)
	err := os.Remove("/port/" + name)
	if err != nil {
		log.Printf("Failed to remove /port/%s: %#v\n", name, err)
	}
}

var myAddress string

// getMyAddress returns a string representing my address from the host's
// point of view. For now this is an IP address but it soon should be a vsock
// port.
func getMyAddress() (string, error) {
	ipv4 := make([]string, 0)
	ipv6 := make([]string, 0)
	for index := 1; ; index++ {
		intf, err := net.InterfaceByIndex(index)
		if err != nil {
			break
		}
		if len(intf.Name) < 3 || intf.Name[0:3] != "eth" {
			continue
		}
		addrs, err := intf.Addrs()
		if err != nil {
			log.Printf("Cannot get addresses for %s: %v", intf.Name, err)
			continue
		}
		for _, addr := range addrs {
			ipAddr, _, err := net.ParseCIDR(addr.String())
			if err != nil {
				continue
			}
			if ipAddr.IsGlobalUnicast() {
				if ipAddr.To4() != nil {
					ip := ipAddr.String()
					ipv4 = append(ipv4, ip)
				} else {
					ip6 := ipAddr.String()
					ipv6 = append(ipv6, ip6)
				}
			}
		}
	}
	// vmnet and hostnet only have IPv4 enabled currently
	if len(ipv4) == 0 {
		log.Println("Unable to find any external IPv4 addresses")
		return "", errors.New("No external IPv4 addresses")
	}
	if len(ipv4) > 1 {
		log.Println("Multiple external IPv4 addresses detected, will choose the first")
	}
	return ipv4[0], nil
}
