package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"libproxy"
	"strings"
)

func main() {
	host, port, container := parseHostContainerAddrs()

	err := exposePort(host, port)
	if err != nil {
		sendError(err)
	}
	p, err := libproxy.NewProxy(host, container)
	if err != nil {
		sendError(err)
	}
	go handleStopSignals(p)
	sendOK()
	p.Run()
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
	_, err = ctl.WriteString(fmt.Sprintf("%s:%08x", name, port))
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
