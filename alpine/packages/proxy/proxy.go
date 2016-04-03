package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"pkg/proxy"
)

// sendError signals the error to the parent and quits the process.
func sendError(err error) {
	f := os.NewFile(3, "signal-parent")

	fmt.Fprintf(f, "1\n%s", err)
	f.Close()
	os.Exit(1)
}

// sendOK signals the parent that the forward is running.
func sendOK() {
	f := os.NewFile(3, "signal-parent")
	fmt.Fprint(f, "0\n")
	f.Close()
}

// From docker/libnetwork/portmapper/proxy.go:

// parseHostContainerAddrs parses the flags passed on reexec to create the TCP or UDP
// net.Addrs to map the host and container ports
func parseHostContainerAddrs() (host net.Addr, port int, container net.Addr) {
	var (
		proto         = flag.String("proto", "tcp", "proxy protocol")
		hostIP        = flag.String("host-ip", "", "host ip")
		hostPort      = flag.Int("host-port", -1, "host port")
		containerIP   = flag.String("container-ip", "", "container ip")
		containerPort = flag.Int("container-port", -1, "container port")
	)

	flag.Parse()

	switch *proto {
	case "tcp":
		host = &net.TCPAddr{IP: net.ParseIP(*hostIP), Port: *hostPort}
		port = *hostPort
		container = &net.TCPAddr{IP: net.ParseIP(*containerIP), Port: *containerPort}
	case "udp":
		host = &net.UDPAddr{IP: net.ParseIP(*hostIP), Port: *hostPort}
		port = *hostPort
		container = &net.UDPAddr{IP: net.ParseIP(*containerIP), Port: *containerPort}
	default:
		log.Fatalf("unsupported protocol %s", *proto)
	}

	return host, port, container
}

func handleStopSignals(p proxy.Proxy) {
	s := make(chan os.Signal, 10)
	signal.Notify(s, os.Interrupt, syscall.SIGTERM, syscall.SIGSTOP)

	for range s {
		p.Close()
	}
}
