package vz

/*
#cgo darwin CFLAGS: -x objective-c -fno-objc-arc
#cgo darwin LDFLAGS: -lobjc -framework Foundation -framework Virtualization
# include "virtualization.h"
*/
import "C"
import "runtime"

// VirtioEntropyDeviceConfiguration is used to expose a source of entropy for the guest operating system’s random-number generator.
// When you create this object and add it to your virtual machine’s configuration, the virtual machine configures a Virtio-compliant
// entropy device. The guest operating system uses this device as a seed to generate random numbers.
//
// see: https://developer.apple.com/documentation/virtualization/vzvirtioentropydeviceconfiguration?language=objc
type VirtioEntropyDeviceConfiguration struct {
	pointer
}

// NewVirtioEntropyDeviceConfiguration creates a new Virtio Entropy Device confiuration.
func NewVirtioEntropyDeviceConfiguration() *VirtioEntropyDeviceConfiguration {
	config := &VirtioEntropyDeviceConfiguration{
		pointer: pointer{
			ptr: C.newVZVirtioEntropyDeviceConfiguration(),
		},
	}
	runtime.SetFinalizer(config, func(self *VirtioEntropyDeviceConfiguration) {
		self.Release()
	})
	return config
}
