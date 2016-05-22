package main

import (
	"encoding/binary"
	"flag"
	"github.com/rneugeba/virtsock/go/vsock"
	"github.com/rneugeba/virtsock/go/hvsock"
	"log"
	"net"
	"proxy/libproxy"
)

// Listen on virtio-vsock and AF_HYPERV for multiplexed connections
func manyPorts() {
	var (
		vsockPort = flag.Int("vsockPort", 62373, "virtio-vsock port")
		hvGuid    = flag.String("hvGuid", "0B95756A-9985-48AD-9470-78E060895BE7", "Hyper-V service GUID")
	)
	flag.Parse()

	listeners := make([]net.Listener, 0)

	vsock, err := vsock.Listen(uint(*vsockPort))
	if err != nil {
		log.Printf("Failed to bind to vsock port %d: %#v", vsockPort, err)
	} else {
		listeners = append(listeners, vsock)
	}
	svcid, _ := hvsock.GuidFromString(*hvGuid)
	hvsock, err := hvsock.Listen(hvsock.HypervAddr{VmId: hvsock.GUID_WILDCARD, ServiceId: svcid})
	if err != nil {
		log.Printf("Failed to bind hvsock guid: %s: %#v", *hvGuid, err)
	} else {
		listeners = append(listeners, hvsock)
	}

	quit := make(chan bool)
	defer close(quit)

	for _, l := range listeners {
		go func(l net.Listener) {
			for {
				conn, err := l.Accept()
				if err != nil {
					log.Printf("Error accepting connection: %#v", err)
					return // no more listening
				}
				go func(conn net.Conn) {
					// Read header which describes TCP/UDP and destination IP:port
					d, err := unmarshalDestination(conn)
					if err != nil {
						log.Printf("Failed to unmarshal header: %#v", err)
						conn.Close()
						return
					}
					switch d.Proto {
					case TCP:
						backendAddr := net.TCPAddr{IP: d.IP, Port: int(d.Port), Zone: ""}
						libproxy.HandleTCPConnection(conn.(libproxy.Conn), &backendAddr, quit)
						break
					case UDP:
						backendAddr := &net.UDPAddr{IP: d.IP, Port: int(d.Port), Zone: ""}

						proxy, err := libproxy.NewUDPProxy(backendAddr, libproxy.NewUDPConn(conn), backendAddr)
						if err != nil {
							log.Printf("Failed to setup UDP proxy for %s: %#v", backendAddr, err)
							conn.Close()
							return
						}
						proxy.Run()
						break
					default:
						log.Printf("Unknown protocol: %d", d.Proto)
						conn.Close()
						return
					}
				}(conn)
			}
		}(l)
	}
	forever := make(chan int)
	<-forever
}

const (
	TCP = 1
	UDP = 2
)

type destination struct {
	Proto uint8
	IP    net.IP
	Port  uint16
}

func unmarshalDestination(conn net.Conn) (destination, error) {
	d := destination{}
	if err := binary.Read(conn, binary.LittleEndian, &d.Proto); err != nil {
		return d, err
	}
	var length uint16
	// IP length
	if err := binary.Read(conn, binary.LittleEndian, &length); err != nil {
		return d, err
	}
	d.IP = make([]byte, length)
	if err := binary.Read(conn, binary.LittleEndian, &d.IP); err != nil {
		return d, err
	}
	if err := binary.Read(conn, binary.LittleEndian, &d.Port); err != nil {
		return d, err
	}
	return d, nil
}
