package dhclient

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/mdlayher/raw"
)

const responseTimeout = time.Second * 5

// Callback is a function called on certain events
type Callback func(*Lease)

// Client is a DHCP client instance
type Client struct {
	Hostname    string
	Iface       *net.Interface
	Lease       *Lease   // The current lease
	OnBound     Callback // On renew or rebound
	OnExpire    Callback // On expiration of a lease
	DHCPOptions []Option // List of options to send on discovery and requests

	conn     *raw.Conn // Raw socket
	xid      uint32    // Transaction ID
	rebind   bool
	shutdown bool
	notify   chan struct{}  // Is closed on shutdown
	wg       sync.WaitGroup // For graceful shutdown
}

// Lease is an assignment by the DHCP server
type Lease struct {
	ServerID     net.IP
	FixedAddress net.IP
	Netmask      net.IPMask
	NextServer   net.IP
	Broadcast    net.IP
	Router       []net.IP
	DNS          []net.IP
	TimeServer   []net.IP
	DomainName   string
	MTU          uint16

	// Other options
	OtherOptions []Option

	Bound  time.Time
	Renew  time.Time
	Rebind time.Time
	Expire time.Time
}

// DefaultParamsRequestList is a list of params to be requested from the server
var DefaultParamsRequestList = []layers.DHCPOpt{
	layers.DHCPOptSubnetMask,   // Subnet Mask
	layers.DHCPOptRouter,       // Router
	layers.DHCPOptTimeServer,   // Time Server
	layers.DHCPOptDNS,          // Domain Name Server
	layers.DHCPOptDomainName,   // Domain Name
	layers.DHCPOptInterfaceMTU, // Interface MTU
	layers.DHCPOptNTPServers,   // Network Time Protocol Servers
}

// AddOption adds an DHCP option
func (client *Client) AddOption(optType layers.DHCPOpt, data []byte) {
	client.DHCPOptions = append(client.DHCPOptions, Option{optType, data})
}

// AddParamRequest adds an parameter to parameter request list, if not included yet.
func (client *Client) AddParamRequest(dhcpOpt layers.DHCPOpt) {

	// search for existing parameter request list
	for i := range client.DHCPOptions {
		if client.DHCPOptions[i].Type == layers.DHCPOptParamsRequest {
			// extend existing list
			client.DHCPOptions[i].AddByte(byte(dhcpOpt))
			return
		}
	}

	// option not added yet
	client.AddOption(layers.DHCPOptParamsRequest, []byte{byte(dhcpOpt)})
}

// Start starts the client
func (client *Client) Start() {

	// Add default DHCP options if none added yet.
	if client.DHCPOptions == nil {
		for _, param := range DefaultParamsRequestList {
			client.AddParamRequest(param)
		}
		client.AddOption(layers.DHCPOptHostname, []byte(client.Hostname))
	}

	if client.notify != nil {
		log.Panicf("client for %s already started", client.Iface.Name)
	}
	client.notify = make(chan struct{})
	client.wg.Add(1)
	go client.run()
}

// Stop stops the client
func (client *Client) Stop() {
	log.Printf("[%s] shutting down dhclient", client.Iface.Name)
	client.shutdown = true
	close(client.notify)

	client.wg.Wait()
}

// Renew triggers the renewal of the current lease
func (client *Client) Renew() {
	select {
	case client.notify <- struct{}{}:
	default:
	}
}

// Rebind forgets the current lease and triggers acquirement of a new one
func (client *Client) Rebind() {
	client.rebind = true
	client.Lease = nil
	client.Renew()
}

func (client *Client) run() {
	for !client.shutdown {
		client.runOnce()
	}
	client.wg.Done()
}

func (client *Client) runOnce() {
	var err error
	if client.Lease == nil || client.rebind {
		// request new lease
		err = client.withConnection(client.discoverAndRequest)
		if err == nil {
			// try to renew the lease in the future
			client.rebind = false
		}
	} else {
		// renew existing lease
		err = client.withConnection(client.renew)
	}

	if err != nil {
		log.Printf("[%s] error: %s", client.Iface.Name, err)
		// delay for a second
		select {
		case <-client.notify:
		case <-time.After(time.Second):
		}
		return
	}

	select {
	case <-client.notify:
		return
	case <-time.After(time.Until(client.Lease.Expire)):
		// remove lease and request a new one
		client.unbound()
	case <-time.After(time.Until(client.Lease.Rebind)):
		// keep lease and request a new one
		client.rebind = true
	case <-time.After(time.Until(client.Lease.Renew)):
		// renew the lease
	}
}

// unbound removes the lease
func (client *Client) unbound() {
	if cb := client.OnExpire; cb != nil {
		cb(client.Lease)
	}
	client.Lease = nil
}

func (client *Client) withConnection(f func() error) error {
	conn, err := raw.ListenPacket(client.Iface, uint16(layers.EthernetTypeIPv4), nil)
	if err != nil {
		return err
	}
	client.conn = conn
	client.xid = rand.Uint32()

	defer func() {
		client.conn.Close()
		client.conn = nil
	}()

	return f()
}

func (client *Client) discoverAndRequest() error {
	lease, err := client.discover()
	if err != nil {
		return err
	}
	return client.request(lease)
}

func (client *Client) renew() error {
	return client.request(client.Lease)
}

func (client *Client) discover() (*Lease, error) {
	err := client.sendPacket(layers.DHCPMsgTypeDiscover, client.DHCPOptions)

	if err != nil {
		return nil, err
	}

	_, lease, err := client.waitForResponse(layers.DHCPMsgTypeOffer)
	if err != nil {
		return nil, err
	}

	return lease, nil
}

func (client *Client) request(lease *Lease) error {
	err := client.sendPacket(layers.DHCPMsgTypeRequest, append(client.DHCPOptions,
		Option{layers.DHCPOptRequestIP, []byte(lease.FixedAddress)},
		Option{layers.DHCPOptServerID, []byte(lease.ServerID)},
	))

	if err != nil {
		return err
	}

	msgType, lease, err := client.waitForResponse(layers.DHCPMsgTypeAck, layers.DHCPMsgTypeNak)
	if err != nil {
		return err
	}

	switch msgType {
	case layers.DHCPMsgTypeAck:
		if lease.Expire.IsZero() {
			err = errors.New("expire value is zero")
			break
		}
		// support DHCP servers that do not send option 58 and 59
		// this is using the Microsoft suggested defaults
		if lease.Renew.IsZero() {
			lease.Renew = lease.Bound.Add(lease.Expire.Sub(lease.Bound) / 2)
		}
		if lease.Rebind.IsZero() {
			lease.Rebind = lease.Bound.Add(lease.Expire.Sub(lease.Bound) / 1000 * 875)
		}

		client.Lease = lease

		// call the handler
		if cb := client.OnBound; cb != nil {
			cb(lease)
		}
	case layers.DHCPMsgTypeNak:
		err = errors.New("received NAK")
		client.unbound()
	default:
		err = fmt.Errorf("unexpected response: %s", msgType.String())
	}

	return err
}

// sendPacket creates and sends a DHCP packet
func (client *Client) sendPacket(msgType layers.DHCPMsgType, options []Option) error {
	log.Printf("[%s] sending %s", client.Iface.Name, msgType)
	return client.sendMulticast(client.newPacket(msgType, options))
}

// newPacket creates a DHCP packet
func (client *Client) newPacket(msgType layers.DHCPMsgType, options []Option) *layers.DHCPv4 {
	packet := layers.DHCPv4{
		Operation:    layers.DHCPOpRequest,
		HardwareType: layers.LinkTypeEthernet,
		ClientHWAddr: client.Iface.HardwareAddr,
		Xid:          client.xid, // Transaction ID
	}

	packet.Options = append(packet.Options, layers.DHCPOption{
		Type:   layers.DHCPOptMessageType,
		Data:   []byte{byte(msgType)},
		Length: 1,
	})

	// append DHCP options
	for _, option := range options {
		packet.Options = append(packet.Options, layers.DHCPOption{
			Type:   option.Type,
			Data:   option.Data,
			Length: uint8(len(option.Data)),
		})
	}

	return &packet
}

func (client *Client) sendMulticast(dhcp *layers.DHCPv4) error {
	eth := layers.Ethernet{
		EthernetType: layers.EthernetTypeIPv4,
		SrcMAC:       client.Iface.HardwareAddr,
		DstMAC:       layers.EthernetBroadcast,
	}
	ip := layers.IPv4{
		Version:  4,
		TTL:      64,
		SrcIP:    []byte{0, 0, 0, 0},
		DstIP:    []byte{255, 255, 255, 255},
		Protocol: layers.IPProtocolUDP,
	}
	udp := layers.UDP{
		SrcPort: 68,
		DstPort: 67,
	}

	// Serialize packet
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}
	udp.SetNetworkLayerForChecksum(&ip)
	err := gopacket.SerializeLayers(buf, opts, &eth, &ip, &udp, dhcp)
	if err != nil {
		return err
	}

	// Send packet
	_, err = client.conn.WriteTo(buf.Bytes(), &raw.Addr{HardwareAddr: eth.DstMAC})
	return err
}

// waitForResponse waits for a DHCP packet with matching transaction ID and the given message type
func (client *Client) waitForResponse(msgTypes ...layers.DHCPMsgType) (layers.DHCPMsgType, *Lease, error) {
	client.conn.SetReadDeadline(time.Now().Add(responseTimeout))

	recvBuf := make([]byte, 1500)
	for {
		_, _, err := client.conn.ReadFrom(recvBuf)

		if err != nil {
			return 0, nil, err
		}

		packet := parsePacket(recvBuf)
		if packet == nil {
			continue
		}

		if packet.Xid == client.xid && packet.Operation == layers.DHCPOpReply {
			msgType, res := newLease(packet)

			// do we have the expected message type?
			for _, t := range msgTypes {
				if t == msgType {
					log.Printf("[%s] received %s", client.Iface.Name, msgType)
					return msgType, &res, nil
				}
			}
		}
	}
}
