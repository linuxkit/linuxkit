package dhclient

import (
	"github.com/google/gopacket/layers"
)

// Option is a DHCP option field
type Option struct {
	Type layers.DHCPOpt
	Data []byte
}

// AddByte ensures a specific byte is included in the data
func (option *Option) AddByte(b byte) {
	for _, o := range option.Data {
		if o == b {
			// already included
			return
		}
	}
	option.Data = append(option.Data, b)
}
