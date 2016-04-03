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
	// TODO: consider whether close/clunk of ctl would be a better tear down
	// signal
	ctl.Close()

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
	if myAddress != "" {
		return myAddress, nil
	}
	d, err := os.Open("/port/docker")
	if err != nil {
		return "", err
	}
	defer d.Close()
	bytes := make([]byte, 100)
	count, err := d.Read(bytes)
	if err != nil {
		return "", err
	}
	s := string(bytes)[0:count]
	bits := strings.Split(s, ":")
	myAddress = bits[2]
	return myAddress, nil
}
