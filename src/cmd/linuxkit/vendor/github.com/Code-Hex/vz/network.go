package vz

/*
#cgo darwin CFLAGS: -x objective-c -fno-objc-arc
#cgo darwin LDFLAGS: -lobjc -framework Foundation -framework Virtualization
# include "virtualization.h"
*/
import "C"
import (
	"net"
	"os"
	"runtime"
)

// BridgedNetwork defines a network interface that bridges a physical interface with a virtual machine.
//
// A bridged interface is shared between the virtual machine and the host system. Both host and
// virtual machine send and receive packets on the same physical interface but have distinct network layers.
//
// The BridgedNetwork can be used with a BridgedNetworkDeviceAttachment to set up a network device NetworkDeviceConfiguration.
// TODO(codehex): implement...
// see: https://developer.apple.com/documentation/virtualization/vzbridgednetworkinterface?language=objc
type BridgedNetwork interface {
	NSObject

	// NetworkInterfaces returns the list of network interfaces available for bridging.
	NetworkInterfaces() []BridgedNetwork

	// Identifier returns the unique identifier for this interface.
	// The identifier is the BSD name associated with the interface (e.g. "en0").
	Identifier() string

	// LocalizedDisplayName returns a display name if available (e.g. "Ethernet").
	LocalizedDisplayName() string
}

// Network device attachment using network address translation (NAT) with outside networks.
//
// Using the NAT attachment type, the host serves as router and performs network address translation
// for accesses to outside networks.
// see: https://developer.apple.com/documentation/virtualization/vznatnetworkdeviceattachment?language=objc
type NATNetworkDeviceAttachment struct {
	pointer

	*baseNetworkDeviceAttachment
}

var _ NetworkDeviceAttachment = (*NATNetworkDeviceAttachment)(nil)

// NewNATNetworkDeviceAttachment creates a new NATNetworkDeviceAttachment.
func NewNATNetworkDeviceAttachment() *NATNetworkDeviceAttachment {
	attachment := &NATNetworkDeviceAttachment{
		pointer: pointer{
			ptr: C.newVZNATNetworkDeviceAttachment(),
		},
	}
	runtime.SetFinalizer(attachment, func(self *NATNetworkDeviceAttachment) {
		self.Release()
	})
	return attachment
}

// BridgedNetworkDeviceAttachment represents a physical interface on the host computer.
//
// Use this struct when configuring a network interface for your virtual machine.
// A bridged network device sends and receives packets on the same physical interface
// as the host computer, but does so using a different network layer.
//
// To use this attachment, your app must have the com.apple.vm.networking entitlement.
// If it doesn’t, the use of this attachment point results in an invalid VZVirtualMachineConfiguration object in objective-c.
//
// see: https://developer.apple.com/documentation/virtualization/vzbridgednetworkdeviceattachment?language=objc
type BridgedNetworkDeviceAttachment struct {
	pointer

	*baseNetworkDeviceAttachment
}

var _ NetworkDeviceAttachment = (*BridgedNetworkDeviceAttachment)(nil)

// NewBridgedNetworkDeviceAttachment creates a new BridgedNetworkDeviceAttachment with networkInterface.
func NewBridgedNetworkDeviceAttachment(networkInterface BridgedNetwork) *BridgedNetworkDeviceAttachment {
	attachment := &BridgedNetworkDeviceAttachment{
		pointer: pointer{
			ptr: C.newVZBridgedNetworkDeviceAttachment(
				networkInterface.Ptr(),
			),
		},
	}
	runtime.SetFinalizer(attachment, func(self *BridgedNetworkDeviceAttachment) {
		self.Release()
	})
	return attachment
}

// FileHandleNetworkDeviceAttachment sending raw network packets over a file handle.
//
// The file handle attachment transmits the raw packets/frames between the virtual network interface and a file handle.
// The data transmitted through this attachment is at the level of the data link layer.
// see: https://developer.apple.com/documentation/virtualization/vzfilehandlenetworkdeviceattachment?language=objc
type FileHandleNetworkDeviceAttachment struct {
	pointer

	*baseNetworkDeviceAttachment
}

var _ NetworkDeviceAttachment = (*FileHandleNetworkDeviceAttachment)(nil)

// NewFileHandleNetworkDeviceAttachment initialize the attachment with a file handle.
//
// file parameter is holding a connected datagram socket.
func NewFileHandleNetworkDeviceAttachment(file *os.File) *FileHandleNetworkDeviceAttachment {
	attachment := &FileHandleNetworkDeviceAttachment{
		pointer: pointer{
			ptr: C.newVZFileHandleNetworkDeviceAttachment(
				C.int(file.Fd()),
			),
		},
	}
	runtime.SetFinalizer(attachment, func(self *FileHandleNetworkDeviceAttachment) {
		self.Release()
	})
	return attachment
}

// NetworkDeviceAttachment for a network device attachment.
// see: https://developer.apple.com/documentation/virtualization/vznetworkdeviceattachment?language=objc
type NetworkDeviceAttachment interface {
	NSObject

	networkDeviceAttachment()
}

type baseNetworkDeviceAttachment struct{}

func (*baseNetworkDeviceAttachment) networkDeviceAttachment() {}

// VirtioNetworkDeviceConfiguration is configuration of a paravirtualized network device of type Virtio Network Device.
//
// The communication channel used on the host is defined through the attachment.
// It is set with the VZNetworkDeviceConfiguration.attachment property in objective-c.
//
// The configuration is only valid with valid MACAddress and attachment.
//
// see: https://developer.apple.com/documentation/virtualization/vzvirtionetworkdeviceconfiguration?language=objc
type VirtioNetworkDeviceConfiguration struct {
	pointer
}

// NewVirtioNetworkDeviceConfiguration creates a new VirtioNetworkDeviceConfiguration with NetworkDeviceAttachment.
func NewVirtioNetworkDeviceConfiguration(attachment NetworkDeviceAttachment) *VirtioNetworkDeviceConfiguration {
	config := &VirtioNetworkDeviceConfiguration{
		pointer: pointer{
			ptr: C.newVZVirtioNetworkDeviceConfiguration(
				attachment.Ptr(),
			),
		},
	}
	runtime.SetFinalizer(config, func(self *VirtioNetworkDeviceConfiguration) {
		self.Release()
	})
	return config
}

func (v *VirtioNetworkDeviceConfiguration) SetMacAddress(macAddress *MACAddress) {
	C.setNetworkDevicesVZMACAddress(v.Ptr(), macAddress.Ptr())
}

// MACAddress represents a media access control address (MAC address), the 48-bit ethernet address.
// see: https://developer.apple.com/documentation/virtualization/vzmacaddress?language=objc
type MACAddress struct {
	pointer
}

// NewMACAddress creates a new MACAddress with net.HardwareAddr (MAC address).
func NewMACAddress(macAddr net.HardwareAddr) *MACAddress {
	macAddrChar := charWithGoString(macAddr.String())
	defer macAddrChar.Free()
	ma := &MACAddress{
		pointer: pointer{
			ptr: C.newVZMACAddress(macAddrChar.CString()),
		},
	}
	runtime.SetFinalizer(ma, func(self *MACAddress) {
		self.Release()
	})
	return ma
}

// NewRandomLocallyAdministeredMACAddress creates a valid, random, unicast, locally administered address.
func NewRandomLocallyAdministeredMACAddress() *MACAddress {
	ma := &MACAddress{
		pointer: pointer{
			ptr: C.newRandomLocallyAdministeredVZMACAddress(),
		},
	}
	runtime.SetFinalizer(ma, func(self *MACAddress) {
		self.Release()
	})
	return ma
}

func (m *MACAddress) String() string {
	cstring := (*char)(C.getVZMACAddressString(m.Ptr()))
	return cstring.String()
}

func (m *MACAddress) HardwareAddr() net.HardwareAddr {
	hw, _ := net.ParseMAC(m.String())
	return hw
}
