package main

// CLI tool for mDNSmon. Monitors a network interface for IP changes and re-publishes the mDNS service

import (
	"flag"
	"log"
	"net"
	"os"
	"syscall"

	"mdnstool/mdnsmon"
)

func main() {

	hostname := flag.String("hostname", "docker.local.", "Hostname - must be FQDN and end with .")
	instance := flag.String("instance", "Moby", "Instance description")
	port := flag.Int("port", 22, "Service port")
	srv := flag.String("service", "_ssh._tcp", "SRV service type")
	info := flag.String("info", "Moby", "TXT service description")
	iface_name := flag.String("if", "eth0", "Network interface to bind multicast listener to. This interface will be monitored for IP changes.")
	detach := flag.Bool("detach", false, "Detach from terminal")

	flag.Parse()

	// Deatch from terminal (based on code from 9pudc)
	if *detach {
		logFile, err := os.Create("/var/log/mdnstool.log")
		if err != nil {
			log.Fatalln("Failed to open log file", err)
		}
		log.SetOutput(logFile)
		null, err := os.OpenFile("/dev/null", os.O_RDWR, 0)
		if err != nil {
			log.Fatalln("Failed to open /dev/null", err)
		}
		fd := null.Fd()
		syscall.Dup2(int(fd), int(os.Stdin.Fd()))
		syscall.Dup2(int(fd), int(os.Stdout.Fd()))
		syscall.Dup2(int(fd), int(os.Stderr.Fd()))
	}

	iface, err := net.InterfaceByName(*iface_name)
	if err != nil {
		log.Fatal(err)
	}

	s, err := mdnsmon.NewServer(*hostname, *instance, *port, *srv, []string{*info}, iface)
	if err != nil {
		log.Fatal(err)
	}

	go s.Start()
	defer s.Shutdown()
	select {}
}
