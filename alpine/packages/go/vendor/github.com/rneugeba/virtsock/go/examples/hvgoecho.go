package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"../hvsock"
)

var (
	clientStr  string
	serverMode bool

	svcid, _ = hvsock.GuidFromString("3049197C-9A4E-4FBF-9367-97F792F16994")
)

func init() {
	flag.StringVar(&clientStr, "c", "", "Client")
	flag.BoolVar(&serverMode, "s", false, "Start as a Server")
}

func server() {
	l, err := hvsock.Listen(hvsock.HypervAddr{VmId: hvsock.GUID_WILDCARD, ServiceId: svcid})
	if err != nil {
		log.Fatalln("Listen():", err)
	}
	defer func() {
		l.Close()
	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatalln("Accept(): ", err)
		}
		fmt.Printf("Received message %s -> %s \n", conn.RemoteAddr(), conn.LocalAddr())

		go handleRequest(conn)
	}
}

func handleRequest(c net.Conn) {
	defer func() {
		fmt.Printf("Closing\n")
		err := c.Close()
		if err != nil {
			log.Fatalln("Close():", err)
		}
	}()

	n, err := io.Copy(c, c)
	if err != nil {
		log.Fatalln("Copy():", err)
	}
	fmt.Printf("Copied Bytes: %d\n", n)

	fmt.Printf("Sending BYE message\n")
	// The '\n' is important as the client use ReadString()
	_, err = fmt.Fprintf(c, "Got %d bytes. Bye\n", n)
	if err != nil {
		log.Fatalln("Failed to send: ", err)
	}
	fmt.Printf("Sent bye\n")
}

func client(vmid hvsock.GUID) {
	sa := hvsock.HypervAddr{VmId: vmid, ServiceId: svcid}
	c, err := hvsock.Dial(sa)
	if err != nil {
		log.Fatalln("Failed to Dial:\n", sa.VmId.String(), sa.ServiceId.String(), err)
	}

	defer func() {
		fmt.Printf("Closing\n")
		c.Close()
	}()

	fmt.Printf("Send: hello\n")
	// Note the '\n' is significant as we use ReadString below
	l, err := fmt.Fprintf(c, "hello\n")
	if err != nil {
		log.Fatalln("Failed to send: ", err)
	}
	fmt.Printf("Sent: %d bytes\n", l)

	message, err := bufio.NewReader(c).ReadString('\n')
	if err != nil {
		log.Fatalln("Failed to receive: ", err)
	}
	fmt.Printf("From SVR: %s", message)

	fmt.Printf("CloseWrite()\n")
	c.CloseWrite()

	fmt.Printf("Waiting for Bye message\n")
	message, err = bufio.NewReader(c).ReadString('\n')
	if err != nil {
		log.Fatalln("Failed to receive: ", err)
	}
	fmt.Printf("From SVR: %s", message)
}

func main() {
	log.SetFlags(log.LstdFlags)
	flag.Parse()

	if serverMode {
		fmt.Printf("Starting server\n")
		server()
	}

	vmid := hvsock.GUID_ZERO
	var err error
	if strings.Contains(clientStr, "-") {
		vmid, err = hvsock.GuidFromString(clientStr)
		if err != nil {
			log.Fatalln("Can't parse GUID: ", clientStr)
		}
	} else if clientStr == "parent" {
		vmid = hvsock.GUID_PARENT
	} else {
		vmid = hvsock.GUID_LOOPBACK
	}
	fmt.Printf("Client connecting to %s", vmid.String())
	client(vmid)
}
