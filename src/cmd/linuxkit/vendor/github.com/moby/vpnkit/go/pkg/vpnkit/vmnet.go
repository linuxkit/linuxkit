package vpnkit

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/google/uuid"
)

// Vmnet describes a "vmnet protocol" connection which allows ethernet frames to be
// sent to and received by vpnkit.
type Vmnet struct {
	conn          net.Conn
	remoteVersion *InitMessage
}

// NewVmnet constructs an instance of Vmnet.
func NewVmnet(ctx context.Context, path string) (*Vmnet, error) {
	d := &net.Dialer{}
	conn, err := d.DialContext(ctx, "unix", path)
	if err != nil {
		return nil, err
	}
	var remoteVersion *InitMessage
	vmnet := &Vmnet{conn, remoteVersion}
	err = vmnet.negotiate()
	if err != nil {
		return nil, err
	}
	return vmnet, err
}

// Close closes the connection.
func (v *Vmnet) Close() error {
	return v.conn.Close()
}

// InitMessage is used for the initial version exchange
type InitMessage struct {
	magic   [5]byte
	version uint32
	commit  [40]byte
}

// String returns a human-readable string.
func (m *InitMessage) String() string {
	return fmt.Sprintf("magic=%v version=%d commit=%v", m.magic, m.version, m.commit)
}

// defaultInitMessage is the init message we will send to vpnkit
func defaultInitMessage() *InitMessage {
	magic := [5]byte{'V', 'M', 'N', '3', 'T'}
	version := uint32(22)
	var commit [40]byte
	copy(commit[:], []byte("0123456789012345678901234567890123456789"))
	return &InitMessage{magic, version, commit}
}

// Write marshals an init message to a connection
func (m *InitMessage) Write(c net.Conn) error {
	if err := binary.Write(c, binary.LittleEndian, m.magic); err != nil {
		return err
	}
	if err := binary.Write(c, binary.LittleEndian, m.version); err != nil {
		return err
	}
	if err := binary.Write(c, binary.LittleEndian, m.commit); err != nil {
		return err
	}
	return nil
}

// readInitMessage unmarshals an init message from a connection
func (v *Vmnet) readInitMessage() (*InitMessage, error) {
	m := defaultInitMessage()
	if err := binary.Read(v.conn, binary.LittleEndian, &m.magic); err != nil {
		return nil, err
	}
	if err := binary.Read(v.conn, binary.LittleEndian, &m.version); err != nil {
		return nil, err
	}
	if err := binary.Read(v.conn, binary.LittleEndian, &m.commit); err != nil {
		return nil, err
	}
	return m, nil
}

func (v *Vmnet) negotiate() error {
	m := defaultInitMessage()
	if err := m.Write(v.conn); err != nil {
		return err
	}
	remoteVersion, err := v.readInitMessage()
	if err != nil {
		return err
	}
	v.remoteVersion = remoteVersion
	return nil
}

// Ethernet requests the creation of a network connection with a given
// uuid and optional IP
type Ethernet struct {
	uuid uuid.UUID
	ip   net.IP
}

// NewEthernet creates an Ethernet frame
func NewEthernet(uuid uuid.UUID, ip net.IP) *Ethernet {
	return &Ethernet{uuid, ip}
}

// Write marshals an Ethernet message
func (m *Ethernet) Write(c net.Conn) error {
	ty := uint8(1)
	if m.ip != nil {
		ty = uint8(8)
	}
	if err := binary.Write(c, binary.LittleEndian, ty); err != nil {
		return err
	}
	u, err := m.uuid.MarshalText()
	if err != nil {
		return err
	}
	if err := binary.Write(c, binary.LittleEndian, u); err != nil {
		return err
	}
	ip := uint32(0)
	if m.ip != nil {
		ip = binary.BigEndian.Uint32(m.ip.To4())
	}
	// The protocol uses little endian, not network endian
	if err := binary.Write(c, binary.LittleEndian, ip); err != nil {
		return err
	}
	return nil
}

// Vif represents an Ethernet device
type Vif struct {
	MTU           uint16
	MaxPacketSize uint16
	ClientMAC     net.HardwareAddr
	IP            net.IP
	conn          net.Conn
}

func (v *Vmnet) readVif() (*Vif, error) {
	var MTU, MaxPacketSize uint16

	if err := binary.Read(v.conn, binary.LittleEndian, &MTU); err != nil {
		return nil, err
	}
	if err := binary.Read(v.conn, binary.LittleEndian, &MaxPacketSize); err != nil {
		return nil, err
	}
	var mac [6]byte
	if err := binary.Read(v.conn, binary.LittleEndian, &mac); err != nil {
		return nil, err
	}
	padding := make([]byte, 1+256-6-2-2)
	if err := binary.Read(v.conn, binary.LittleEndian, &padding); err != nil {
		return nil, err
	}
	ClientMAC := mac[:]
	conn := v.conn
	var IP net.IP
	return &Vif{MTU, MaxPacketSize, ClientMAC, IP, conn}, nil
}

// ConnectVif returns a connected network interface with the given uuid.
func (v *Vmnet) ConnectVif(uuid uuid.UUID) (*Vif, error) {
	e := NewEthernet(uuid, nil)
	if err := e.Write(v.conn); err != nil {
		return nil, err
	}
	var responseType uint8
	if err := binary.Read(v.conn, binary.LittleEndian, &responseType); err != nil {
		return nil, err
	}
	switch responseType {
	case 1:
		vif, err := v.readVif()
		if err != nil {
			return nil, err
		}
		IP, err := vif.dhcp()
		if err != nil {
			return nil, err
		}
		vif.IP = IP
		return vif, err
	default:
		var len uint8
		if err := binary.Read(v.conn, binary.LittleEndian, &len); err != nil {
			return nil, err
		}
		message := make([]byte, len)
		if err := binary.Read(v.conn, binary.LittleEndian, &message); err != nil {
			return nil, err
		}
		return nil, errors.New(string(message))
	}
}

// ConnectVifIP returns a connected network interface with the given uuid
// and IP. If the IP is already in use then return an error.
func (v *Vmnet) ConnectVifIP(uuid uuid.UUID, IP net.IP) (*Vif, error) {
	e := NewEthernet(uuid, IP)
	if err := e.Write(v.conn); err != nil {
		return nil, err
	}
	var responseType uint8
	if err := binary.Read(v.conn, binary.LittleEndian, &responseType); err != nil {
		return nil, err
	}
	switch responseType {
	case 1:
		vif, err := v.readVif()
		if err != nil {
			return nil, err
		}
		vif.IP = IP
		return vif, err
	default:
		var len uint8
		if err := binary.Read(v.conn, binary.LittleEndian, &len); err != nil {
			return nil, err
		}
		message := make([]byte, len)
		if err := binary.Read(v.conn, binary.LittleEndian, &message); err != nil {
			return nil, err
		}
		return nil, errors.New(string(message))
	}
}

// Write writes a packet to a Vif
func (v *Vif) Write(packet []byte) error {
	len := uint16(len(packet))
	if err := binary.Write(v.conn, binary.LittleEndian, len); err != nil {
		return err
	}
	if err := binary.Write(v.conn, binary.LittleEndian, packet); err != nil {
		return err
	}
	return nil
}

// Read reads the next packet from a Vif
func (v *Vif) Read() ([]byte, error) {
	var len uint16
	if err := binary.Read(v.conn, binary.LittleEndian, &len); err != nil {
		return nil, err
	}
	packet := make([]byte, len)
	if err := binary.Read(v.conn, binary.LittleEndian, &packet); err != nil {
		return nil, err
	}
	return packet, nil
}

// PcapWriter writes pcap-formatted packet streams
type PcapWriter struct {
	w       io.Writer
	snaplen uint32
}

// NewPcapWriter creates a PcapWriter and writes the initial header
func NewPcapWriter(w io.Writer) (*PcapWriter, error) {
	magic := uint32(0xa1b2c3d4)
	major := uint16(2)
	minor := uint16(4)
	thiszone := uint32(0)   // GMT to local correction
	sigfigs := uint32(0)    // accuracy of local timestamps
	snaplen := uint32(1500) // max length of captured packets, in octets
	network := uint32(1)    // ethernet
	if err := binary.Write(w, binary.LittleEndian, magic); err != nil {
		return nil, err
	}
	if err := binary.Write(w, binary.LittleEndian, major); err != nil {
		return nil, err
	}
	if err := binary.Write(w, binary.LittleEndian, minor); err != nil {
		return nil, err
	}
	if err := binary.Write(w, binary.LittleEndian, thiszone); err != nil {
		return nil, err
	}
	if err := binary.Write(w, binary.LittleEndian, sigfigs); err != nil {
		return nil, err
	}
	if err := binary.Write(w, binary.LittleEndian, snaplen); err != nil {
		return nil, err
	}
	if err := binary.Write(w, binary.LittleEndian, network); err != nil {
		return nil, err
	}
	return &PcapWriter{w, snaplen}, nil
}

// Write appends a packet with a pcap-format header
func (p *PcapWriter) Write(packet []byte) error {
	stamp := time.Now()
	s := uint32(stamp.Second())
	us := uint32(stamp.Nanosecond() / 1000)
	actualLen := uint32(len(packet))
	if err := binary.Write(p.w, binary.LittleEndian, s); err != nil {
		return err
	}
	if err := binary.Write(p.w, binary.LittleEndian, us); err != nil {
		return err
	}
	toWrite := packet[:]
	if actualLen > p.snaplen {
		toWrite = toWrite[0:p.snaplen]
	}
	caplen := uint32(len(toWrite))
	if err := binary.Write(p.w, binary.LittleEndian, caplen); err != nil {
		return err
	}
	if err := binary.Write(p.w, binary.LittleEndian, actualLen); err != nil {
		return err
	}

	if err := binary.Write(p.w, binary.LittleEndian, toWrite); err != nil {
		return err
	}
	return nil
}

// EthernetFrame is an ethernet frame
type EthernetFrame struct {
	Dst  net.HardwareAddr
	Src  net.HardwareAddr
	Type uint16
	Data []byte
}

// NewEthernetFrame constructs an Ethernet frame
func NewEthernetFrame(Dst, Src net.HardwareAddr, Type uint16) *EthernetFrame {
	Data := make([]byte, 0)
	return &EthernetFrame{Dst, Src, Type, Data}
}

func (e *EthernetFrame) setData(data []byte) {
	e.Data = data
}

// Write marshals an Ethernet frame
func (e *EthernetFrame) Write(w io.Writer) error {
	if err := binary.Write(w, binary.BigEndian, e.Dst); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, e.Src); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, e.Type); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, e.Data); err != nil {
		return err
	}
	return nil
}

// ParseEthernetFrame parses the ethernet frame
func ParseEthernetFrame(frame []byte) (*EthernetFrame, error) {
	if len(frame) < (6 + 6 + 2) {
		return nil, errors.New("Ethernet frame is too small")
	}
	Dst := frame[0:6]
	Src := frame[6:12]
	Type := uint16(frame[12])<<8 + uint16(frame[13])
	Data := frame[14:]
	return &EthernetFrame{Dst, Src, Type, Data}, nil
}

// Bytes returns the marshalled ethernet frame
func (e *EthernetFrame) Bytes() []byte {
	buf := bytes.NewBufferString("")
	if err := e.Write(buf); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// Ipv4 is an IPv4 frame
type Ipv4 struct {
	Dst      net.IP
	Src      net.IP
	Data     []byte
	Checksum uint16
}

// NewIpv4 constructs a new empty IPv4 packet
func NewIpv4(Dst, Src net.IP) *Ipv4 {
	Checksum := uint16(0)
	Data := make([]byte, 0)
	return &Ipv4{Dst, Src, Data, Checksum}
}

// ParseIpv4 parses an IP packet
func ParseIpv4(packet []byte) (*Ipv4, error) {
	if len(packet) < 20 {
		return nil, errors.New("IPv4 packet too small")
	}
	ihl := int((packet[0] & 0xf) * 4) // in octets
	if len(packet) < ihl {
		return nil, errors.New("IPv4 packet too small")
	}
	Dst := packet[12:16]
	Src := packet[16:20]
	Data := packet[ihl:]
	Checksum := uint16(0) // assume offload
	return &Ipv4{Dst, Src, Data, Checksum}, nil
}

func (i *Ipv4) setData(data []byte) {
	i.Data = data
	i.Checksum = uint16(0) // as if we were using offload
}

// HeaderBytes returns the marshalled form of the IPv4 header
func (i *Ipv4) HeaderBytes() []byte {
	len := len(i.Data) + 20
	length := [2]byte{byte(len >> 8), byte(len & 0xff)}
	checksum := [2]byte{byte(i.Checksum >> 8), byte(i.Checksum & 0xff)}
	return []byte{
		0x45,                 // version + IHL
		0x00,                 // DSCP + ECN
		length[0], length[1], // total length
		0x7f, 0x61, // Identification
		0x00, 0x00, // Flags + Fragment offset
		0x40, // TTL
		0x11, // Protocol
		checksum[0], checksum[1],
		0x00, 0x00, 0x00, 0x00, // source
		0xff, 0xff, 0xff, 0xff, // destination
	}
}

// Bytes returns the marshalled IPv4 packet
func (i *Ipv4) Bytes() []byte {
	header := i.HeaderBytes()
	return append(header, i.Data...)
}

// Udpv4 is a Udpv4 frame
type Udpv4 struct {
	Src      uint16
	Dst      uint16
	Data     []byte
	Checksum uint16
}

// NewUdpv4 constructs a Udpv4 frame
func NewUdpv4(ipv4 *Ipv4, Dst, Src uint16, Data []byte) *Udpv4 {
	Checksum := uint16(0)
	return &Udpv4{Dst, Src, Data, Checksum}
}

// ParseUdpv4 parses a Udpv4 packet
func ParseUdpv4(packet []byte) (*Udpv4, error) {
	if len(packet) < 8 {
		return nil, errors.New("UDPv4 is too short")
	}
	Src := uint16(packet[0])<<8 + uint16(packet[1])
	Dst := uint16(packet[2])<<8 + uint16(packet[3])
	Checksum := uint16(packet[6])<<8 + uint16(packet[7])
	Data := packet[8:]
	return &Udpv4{Src, Dst, Data, Checksum}, nil
}

// Write marshalls a Udpv4 frame
func (u *Udpv4) Write(w io.Writer) error {
	if err := binary.Write(w, binary.BigEndian, u.Src); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, u.Dst); err != nil {
		return err
	}
	length := uint16(8 + len(u.Data))
	if err := binary.Write(w, binary.BigEndian, length); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, u.Checksum); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, u.Data); err != nil {
		return err
	}
	return nil
}

// Bytes returns the marshalled Udpv4 frame
func (u *Udpv4) Bytes() []byte {
	buf := bytes.NewBufferString("")
	if err := u.Write(buf); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// DhcpRequest is a simple DHCP request
type DhcpRequest struct {
	MAC net.HardwareAddr
}

// NewDhcpRequest constructs a DHCP request
func NewDhcpRequest(MAC net.HardwareAddr) *DhcpRequest {
	if len(MAC) != 6 {
		panic("MAC address must be 6 bytes")
	}
	return &DhcpRequest{MAC}
}

// Bytes returns the marshalled DHCP request
func (d *DhcpRequest) Bytes() []byte {
	bs := []byte{
		0x01,                   // OP
		0x01,                   // HTYPE
		0x06,                   // HLEN
		0x00,                   // HOPS
		0x01, 0x00, 0x00, 0x00, // XID
		0x00, 0x00, // SECS
		0x80, 0x00, // FLAGS
		0x00, 0x00, 0x00, 0x00, // CIADDR
		0x00, 0x00, 0x00, 0x00, // YIADDR
		0x00, 0x00, 0x00, 0x00, // SIADDR
		0x00, 0x00, 0x00, 0x00, // GIADDR
		d.MAC[0], d.MAC[1], d.MAC[2], d.MAC[3], d.MAC[4], d.MAC[5],
	}
	bs = append(bs, make([]byte, 202)...)
	bs = append(bs, []byte{
		0x63, 0x82, 0x53, 0x63, // Magic cookie
		0x35, 0x01, 0x01, // DHCP discover
		0xff, // Endmark
	}...)
	return bs
}

// dhcp queries the IP by DHCP
func (v *Vif) dhcp() (net.IP, error) {
	broadcastMAC := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	broadcastIP := []byte{0xff, 0xff, 0xff, 0xff}
	unknownIP := []byte{0, 0, 0, 0}

	dhcpRequest := NewDhcpRequest(v.ClientMAC).Bytes()
	ipv4 := NewIpv4(broadcastIP, unknownIP)

	udpv4 := NewUdpv4(ipv4, 68, 67, dhcpRequest)
	ipv4.setData(udpv4.Bytes())

	ethernet := NewEthernetFrame(broadcastMAC, v.ClientMAC, 0x800)
	ethernet.setData(ipv4.Bytes())

	file, err := os.Create("/tmp/go.pcap")
	if err != nil {
		panic(err)
	}
	pcap, err := NewPcapWriter(file)
	if err != nil {
		panic(err)
	}
	finished := false
	go func() {
		for !finished {
			if err := v.Write(ethernet.Bytes()); err != nil {
				panic(err)
			}
			if err := pcap.Write(ethernet.Bytes()); err != nil {
				panic(err)
			}
			time.Sleep(time.Second)
		}
	}()

	for {
		response, err := v.Read()
		if err != nil {
			return nil, err
		}
		if err := pcap.Write(response); err != nil {
			panic(err)
		}
		ethernet, err = ParseEthernetFrame(response)
		if err != nil {
			continue
		}
		for i, x := range ethernet.Dst {
			if i > len(v.ClientMAC) || v.ClientMAC[i] != x {
				// intended for someone else
				continue
			}
		}
		ipv4, err = ParseIpv4(ethernet.Data)
		if err != nil {
			// probably not an IPv4 packet
			continue
		}
		udpv4, err = ParseUdpv4(ipv4.Data)
		if err != nil {
			// probably not a UDPv4 packet
			continue
		}
		if udpv4.Src != 67 || udpv4.Dst != 68 {
			// not a DHCP response
			continue
		}
		if len(udpv4.Data) < 243 {
			// truncated
			continue
		}
		if udpv4.Data[240] != 53 || udpv4.Data[241] != 1 || udpv4.Data[242] != 2 {
			// not a DHCP offer
			continue
		}
		var ip net.IP
		ip = udpv4.Data[16:20]
		finished = true // will terminate sending goroutine
		return ip, nil
	}

}
