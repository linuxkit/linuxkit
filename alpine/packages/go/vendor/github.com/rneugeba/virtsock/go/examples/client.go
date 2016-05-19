package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"

	"../"
)

var (
	vmstr   string
	portstr string
)

func init() {
	flag.StringVar(&vmstr, "vm", "", "Hyper-V VM to connect to")
	flag.StringVar(&portstr, "port", "23a432c2-537a-4291-bcb5-d62504644739", "Hyper-V sockets service/port")
}

func main() {
	log.SetFlags(log.LstdFlags)
	flag.Parse()

	vmid, err := hvsock.GuidFromString(vmstr)
	if err != nil {
		log.Fatalln("Failed to parse GUID", vmstr, err)
	}
	svcid, err := hvsock.GuidFromString(portstr)
	if err != nil {
		log.Fatalln("Failed to parse GUID", portstr, err)
	}

	c, err := hvsock.Dial(hvsock.HypervAddr{VmId: vmid, ServiceId: svcid})
	if err != nil {
		log.Fatalln("Failed to Dial:\n", vmstr, portstr, err)
	}

	fmt.Println("Send: hello")
	l, err := fmt.Fprintf(c, "hello\n")
	if err != nil {
		log.Fatalln("Failed to send: ", err)
	}
	fmt.Println("Sent: %s bytes", l)

	message, _ := bufio.NewReader(c).ReadString('\n')
	fmt.Println("From SVR: " + message)
}
