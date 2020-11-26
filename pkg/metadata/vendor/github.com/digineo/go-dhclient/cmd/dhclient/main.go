package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/digineo/go-dhclient"
	"github.com/google/gopacket/layers"
)

var (
	options       = optionList{}
	requestParams = byteList{}
)

func init() {
	flag.Usage = func() {
		fmt.Printf("syntax: %s [flags] IFNAME\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Var(&options, "option", "custom DHCP option for the request (code,value)")
	flag.Var(&requestParams, "request", "Additional value for the DHCP Request List Option 55 (code)")
}

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	ifname := flag.Arg(0)
	iface, err := net.InterfaceByName(ifname)
	if err != nil {
		fmt.Printf("unable to find interface %s: %s\n", ifname, err)
		os.Exit(1)
	}

	client := dhclient.Client{
		Iface: iface,
		OnBound: func(lease *dhclient.Lease) {
			log.Printf("Bound: %+v", lease)
		},
	}

	// Add requests for default options
	for _, param := range dhclient.DefaultParamsRequestList {
		log.Printf("Requesting default option %d", param)
		client.AddParamRequest(layers.DHCPOpt(param))
	}

	// Add requests for custom options
	for _, param := range requestParams {
		log.Printf("Requesting custom option %d", param)
		client.AddParamRequest(layers.DHCPOpt(param))
	}

	// Add hostname option
	hostname, _ := os.Hostname()
	client.AddOption(layers.DHCPOptHostname, []byte(hostname))

	// Add custom options
	for _, option := range options {
		log.Printf("Adding option %d=0x%x", option.Type, option.Data)
		client.AddOption(option.Type, option.Data)
	}

	client.Start()
	defer client.Stop()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGUSR1)
	for {
		sig := <-c
		log.Println("received", sig)
		switch sig {
		case syscall.SIGINT, syscall.SIGTERM:
			return
		case syscall.SIGHUP:
			log.Println("renew lease")
			client.Renew()
		case syscall.SIGUSR1:
			log.Println("acquire new lease")
			client.Rebind()
		}
	}
}
