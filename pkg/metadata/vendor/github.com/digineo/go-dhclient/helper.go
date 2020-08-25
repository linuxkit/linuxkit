package dhclient

import (
	"encoding/binary"
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// parseIPs slices the data into net.IP pieces of 4 bytes
func parseIPs(data []byte) []net.IP {
	result := make([]net.IP, len(data)/4)
	for i := 0; i+3 < len(data); i += 4 {
		result[i/4] = net.IP(data[i : i+4])
	}
	return result
}

// parsePacket decodes a DHCPv4 packet
func parsePacket(data []byte) *layers.DHCPv4 {
	packet := gopacket.NewPacket(data, layers.LayerTypeEthernet, gopacket.Default)
	dhcpLayer := packet.Layer(layers.LayerTypeDHCPv4)

	if dhcpLayer == nil {
		// received packet is not DHCP
		return nil
	}
	return dhcpLayer.(*layers.DHCPv4)
}

// newLease transforms a DHCP offer into a Lease
func newLease(packet *layers.DHCPv4) (msgType layers.DHCPMsgType, lease Lease) {
	lease.Bound = time.Now()
	lease.FixedAddress = packet.YourClientIP

	for _, option := range packet.Options {
		switch option.Type {
		case layers.DHCPOptMessageType:
			if option.Length == 1 {
				msgType = layers.DHCPMsgType(option.Data[0])
			}
		case layers.DHCPOptSubnetMask:
			lease.Netmask = net.IPMask(option.Data)
		case layers.DHCPOptBroadcastAddr:
			lease.Broadcast = net.IP(option.Data)
		case layers.DHCPOptServerID:
			lease.ServerID = net.IP(option.Data)
		case layers.DHCPOptRouter:
			lease.Router = parseIPs(option.Data)
		case layers.DHCPOptDNS:
			lease.DNS = parseIPs(option.Data)
		case layers.DHCPOptTimeServer:
			lease.TimeServer = parseIPs(option.Data)
		case layers.DHCPOptDomainName:
			lease.DomainName = string(option.Data)
		case layers.DHCPOptInterfaceMTU:
			if option.Length == 2 {
				lease.MTU = binary.BigEndian.Uint16(option.Data)
			}
		case layers.DHCPOptLeaseTime:
			if option.Length == 4 {
				lease.Expire = lease.Bound.Add(time.Second * time.Duration(binary.BigEndian.Uint32(option.Data)))
			}
		case layers.DHCPOptT1:
			if option.Length == 4 {
				lease.Renew = lease.Bound.Add(time.Second * time.Duration(binary.BigEndian.Uint32(option.Data)))
			}
		case layers.DHCPOptT2:
			if option.Length == 4 {
				lease.Rebind = lease.Bound.Add(time.Second * time.Duration(binary.BigEndian.Uint32(option.Data)))
			}
		default:
			lease.OtherOptions = append(lease.OtherOptions, Option{option.Type, option.Data})
		}
	}
	return
}
