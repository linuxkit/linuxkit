package vz

/*
#cgo darwin CFLAGS: -x objective-c -fno-objc-arc
#cgo darwin LDFLAGS: -lobjc -framework Foundation -framework Virtualization
# include "virtualization.h"
*/
import "C"
import "runtime"

// MemoryBalloonDeviceConfiguration for a memory balloon device configuration.
type MemoryBalloonDeviceConfiguration interface {
	NSObject

	memoryBalloonDeviceConfiguration()
}

type baseMemoryBalloonDeviceConfiguration struct{}

func (*baseMemoryBalloonDeviceConfiguration) memoryBalloonDeviceConfiguration() {}

var _ MemoryBalloonDeviceConfiguration = (*VirtioTraditionalMemoryBalloonDeviceConfiguration)(nil)

// VirtioTraditionalMemoryBalloonDeviceConfiguration is a configuration of the Virtio traditional memory balloon device.
//
// see: https://developer.apple.com/documentation/virtualization/vzvirtiotraditionalmemoryballoondeviceconfiguration?language=objc
type VirtioTraditionalMemoryBalloonDeviceConfiguration struct {
	pointer

	*baseMemoryBalloonDeviceConfiguration
}

// NewVirtioTraditionalMemoryBalloonDeviceConfiguration creates a new VirtioTraditionalMemoryBalloonDeviceConfiguration.
func NewVirtioTraditionalMemoryBalloonDeviceConfiguration() *VirtioTraditionalMemoryBalloonDeviceConfiguration {
	config := &VirtioTraditionalMemoryBalloonDeviceConfiguration{
		pointer: pointer{
			ptr: C.newVZVirtioTraditionalMemoryBalloonDeviceConfiguration(),
		},
	}
	runtime.SetFinalizer(config, func(self *VirtioTraditionalMemoryBalloonDeviceConfiguration) {
		self.Release()
	})
	return config
}
