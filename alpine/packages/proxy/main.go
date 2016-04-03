package main

import (
	"log"
	"net"
	"os"
	"pkg/proxy"
)

func main() {
	host, container := parseHostContainerAddrs()

	err := exposePort(host)
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

func exposePort(host net.Addr) error {
	log.Printf("exposePort %#v\n", host)
	return nil
}

func unexposePort(host net.Addr) {
	log.Printf("unexposePort %#v\n", host)
}
