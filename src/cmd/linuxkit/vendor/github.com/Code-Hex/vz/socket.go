package vz

/*
#cgo darwin CFLAGS: -x objective-c -fno-objc-arc
#cgo darwin LDFLAGS: -lobjc -framework Foundation -framework Virtualization
# include "virtualization.h"
*/
import "C"
import "runtime"

// SocketDeviceConfiguration for a socket device configuration.
type SocketDeviceConfiguration interface {
	NSObject

	socketDeviceConfiguration()
}

type baseSocketDeviceConfiguration struct{}

func (*baseSocketDeviceConfiguration) socketDeviceConfiguration() {}

var _ SocketDeviceConfiguration = (*VirtioSocketDeviceConfiguration)(nil)

// VirtioSocketDeviceConfiguration is a configuration of the Virtio socket device.
//
// This configuration creates a Virtio socket device for the guest which communicates with the host through the Virtio interface.
// Only one Virtio socket device can be used per virtual machine.
// see: https://developer.apple.com/documentation/virtualization/vzvirtiosocketdeviceconfiguration?language=objc
type VirtioSocketDeviceConfiguration struct {
	pointer

	*baseSocketDeviceConfiguration
}

// NewVirtioSocketDeviceConfiguration creates a new VirtioSocketDeviceConfiguration.
func NewVirtioSocketDeviceConfiguration() *VirtioSocketDeviceConfiguration {
	config := &VirtioSocketDeviceConfiguration{
		pointer: pointer{
			ptr: C.newVZVirtioSocketDeviceConfiguration(),
		},
	}
	runtime.SetFinalizer(config, func(self *VirtioSocketDeviceConfiguration) {
		self.Release()
	})
	return config
}
