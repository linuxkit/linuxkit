package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"proxy/libproxy"
	"strings"
	"github.com/djs55/vsock"
)

func main() {
	host, port, container := parseHostContainerAddrs()

	vsockP, err := libproxy.NewVsockProxy(&vsock.VsockAddr{Port: uint(port)}, container)
	if err != nil {
		sendError(err)
	}
	ipP, err := libproxy.NewIPProxy(host, container)
	if err != nil {
		sendError(err)
	}

	ctl, err := exposePort(host, port)
	if err != nil {
		sendError(err)
	}

	go handleStopSignals(ipP)
	// TODO: avoid this line if we are running in a TTY
	sendOK()
	go ipP.Run()
	vsockP.Run()
	ctl.Close() // ensure ctl remains alive and un-GCed until here
	os.Exit(0)
}

func exposePort(host net.Addr, port int) (*os.File, error) {
	name := host.Network() + ":" + host.String()
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
	_, err = ctl.WriteString(fmt.Sprintf("%s:%08x", name, port))
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
