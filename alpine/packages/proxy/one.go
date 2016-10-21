package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"proxy/libproxy"
	"strings"
	"syscall"
)

func onePort() {
	host, _, container, localIP := parseHostContainerAddrs()

	var ipP libproxy.Proxy
	var err error

	if localIP {
		ipP, err = listenInVM(host, container)
		if err != nil {
			sendError(err)
		}
	}

	ctl, err := exposePort(host, container)
	if err != nil {
		sendError(err)
	}

	go handleStopSignals()
	// TODO: avoid this line if we are running in a TTY
	sendOK()
	if ipP != nil {
		ipP.Run()
	} else {
		select {} // sleep forever
	}
	ctl.Close() // ensure ctl remains alive and un-GCed until here
	os.Exit(0)
}

// Best-effort attempt to listen on the address in the VM. This is for
// backwards compatibility with software that expects to be able to listen on
// 0.0.0.0 and then connect from within a container to the external port.
// If the address doesn't exist in the VM (i.e. it exists only on the host)
// then this is not a hard failure.
func listenInVM(host net.Addr, container net.Addr) (libproxy.Proxy, error) {
	ipP, err := libproxy.NewIPProxy(host, container)
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

func exposePort(host net.Addr, container net.Addr) (*os.File, error) {
	name := host.Network() + ":" + host.String() + ":" + container.Network() + ":" + container.String()
	log.Printf("exposePort %s\n", name)
	err := os.Mkdir("/port/"+name, 0)
	if err != nil {
		log.Printf("Failed to mkdir /port/%s: %#v\n", name, err)
		return nil, err
	}
	ctl, err := os.OpenFile("/port/"+name+"/ctl", os.O_RDWR, 0)
	if err != nil {
		log.Printf("Failed to open /port/%s/ctl: %#v\n", name, err)
		return nil, err
	}
	_, err = ctl.WriteString(fmt.Sprintf("%s", name))
	if err != nil {
		log.Printf("Failed to open /port/%s/ctl: %#v\n", name, err)
		return nil, err
	}
	_, err = ctl.Seek(0, 0)
	if err != nil {
		log.Printf("Failed to seek on /port/%s/ctl: %#v\n", name, err)
		return nil, err
	}
	results := make([]byte, 100)
	count, err := ctl.Read(results)
	if err != nil {
		log.Printf("Failed to read from /port/%s/ctl: %#v\n", name, err)
		return nil, err
	}
	// We deliberately keep the control file open since 9P clunk
	// will trigger a shutdown on the host side.

	response := string(results[0:count])
	if strings.HasPrefix(response, "ERROR ") {
		os.Remove("/port/" + name + "/ctl")
		response = strings.Trim(response[6:], " \t\r\n")
		return nil, errors.New(response)
	}
	// Hold on to a reference to prevent premature GC and close
	return ctl, nil
}
