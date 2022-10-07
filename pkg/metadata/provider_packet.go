package main

import (
	"fmt"
	"net"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/packethost/packngo/metadata"
	"github.com/vishvananda/netlink"
)

// ProviderPacket is the type implementing the Provider interface for Packet.net
type ProviderPacket struct {
	metadata *metadata.CurrentDevice
	err      error
}

// NewPacket returns a new ProviderPacket
func NewPacket() *ProviderPacket {
	return &ProviderPacket{}
}

func (p *ProviderPacket) String() string {
	return "Packet"
}

// Probe checks if we are running on Packet
func (p *ProviderPacket) Probe() bool {
	// Unfortunately the host is resolveable globally, so no easy test
	p.metadata, p.err = metadata.GetMetadata()
	return p.err == nil
}

// Extract gets both the Packet specific and generic userdata
func (p *ProviderPacket) Extract() ([]byte, error) {
	// do not retrieve if we Probed
	if p.metadata == nil && p.err == nil {
		p.metadata, p.err = metadata.GetMetadata()
		if p.err != nil {
			return nil, p.err
		}
	} else if p.err != nil {
		return nil, p.err
	}

	if err := os.WriteFile(path.Join(ConfigPath, Hostname), []byte(p.metadata.Hostname), 0644); err != nil {
		return nil, fmt.Errorf("Packet: Failed to write hostname: %s", err)
	}

	if err := os.MkdirAll(path.Join(ConfigPath, SSH), 0755); err != nil {
		return nil, fmt.Errorf("Failed to create %s: %s", SSH, err)
	}

	sshKeys := strings.Join(p.metadata.SSHKeys, "\n")

	if err := os.WriteFile(path.Join(ConfigPath, SSH, "authorized_keys"), []byte(sshKeys), 0600); err != nil {
		return nil, fmt.Errorf("Failed to write ssh keys: %s", err)
	}

	if err := networkConfig(p.metadata.Network); err != nil {
		return nil, err
	}

	userData, err := metadata.GetUserData()
	if err != nil {
		return nil, fmt.Errorf("Packet: failed to get userdata: %s", err)
	}

	if len(userData) == 0 {
		return nil, nil
	}

	if len(userData) > 6 && string(userData[0:6]) == "#!ipxe" {
		// if you use the userdata for ipxe boot, no use as userdata
		return nil, nil
	}

	return userData, nil
}

// networkConfig handles Packet network configuration, primarily bonding
func networkConfig(ni metadata.NetworkInfo) error {
	// rename interfaces to match what the metadata calls them
	links, err := netlink.LinkList()
	if err != nil {
		return fmt.Errorf("Failed to list links: %v", err)
	}
	for _, link := range links {
		attrs := link.Attrs()
		mac := attrs.HardwareAddr.String()
		for _, iface := range ni.Interfaces {
			if iface.MAC == mac {
				// remove existing addresses from link
				addresses, err := netlink.AddrList(link, netlink.FAMILY_ALL)
				if err != nil {
					return fmt.Errorf("Cannot list addresses on interface: %v", err)
				}
				for _, addr := range addresses {
					if err := netlink.AddrDel(link, &addr); err != nil {
						return fmt.Errorf("Cannot remove address from interface: %v", err)
					}
				}
				// links need to be down to be bonded
				if err := netlink.LinkSetDown(link); err != nil {
					return fmt.Errorf("Cannot down link: %v", err)
				}
				// see if we need to rename interface
				if iface.Name != attrs.Name {
					if err := netlink.LinkSetName(link, iface.Name); err != nil {
						return fmt.Errorf("Interface rename failed: %v", err)
					}
				}
			}
		}
	}

	// set up bonding
	la := netlink.LinkAttrs{Name: "bond0"}
	bond := &netlink.GenericLink{la, "bond"}
	if err := netlink.LinkAdd(bond); err != nil {
		// weirdly creating a bind always seems to return EEXIST
		fmt.Fprintf(os.Stderr, "Error adding bond0: %v (ignoring)", err)
	}
	if err := os.WriteFile("/sys/class/net/bond0/bonding/mode", []byte(strconv.Itoa(int(ni.Bonding.Mode))), 0); err != nil {
		return fmt.Errorf("Cannot write to /sys/class/net/bond0/bonding/mode: %v", err)
	}
	if err := netlink.LinkSetUp(bond); err != nil {
		return fmt.Errorf("Failed to bring bond0 up: %v", err)
	}
	for _, iface := range ni.Interfaces {
		link, err := netlink.LinkByName(iface.Name)
		if err != nil {
			return fmt.Errorf("Cannot find interface %s: %v", iface.Name, err)
		}
		if err := netlink.LinkSetMasterByIndex(link, bond.Attrs().Index); err != nil {
			return fmt.Errorf("Cannot join %s to bond0: %v", iface.Name, err)
		}
	}

	// set up addresses and routes
	for _, address := range ni.Addresses {
		cidr := "/" + strconv.Itoa(address.NetworkBits)
		addr, err := netlink.ParseAddr(address.Address.String() + cidr)
		if err != nil {
			return err
		}
		if err := netlink.AddrAdd(bond, addr); err != nil {
			return fmt.Errorf("Failed to add address to bonded interface: %v", err)
		}
		var additionalNet string
		switch {
		case address.Public && address.Family == metadata.IPv4:
			additionalNet = "0.0.0.0/0"
		case address.Public && address.Family == metadata.IPv6:
			additionalNet = "::/0"
		case !address.Public && address.Family == metadata.IPv4:
			// add a gateway for eg 10.0.0.0/8
			mask := address.Address.DefaultMask()
			masked := address.Address.Mask(mask)
			network := &net.IPNet{IP: masked, Mask: mask}
			additionalNet = network.String()
		}
		if additionalNet != "" {
			_, network, err := net.ParseCIDR(additionalNet)
			if err != nil {
				return err
			}
			route := netlink.Route{
				LinkIndex: bond.Attrs().Index,
				Gw:        address.Gateway,
				Dst:       network,
			}
			if err := netlink.RouteAdd(&route); err != nil {
				return fmt.Errorf("Failed to add route: %v", err)
			}
		}
	}

	return nil
}
