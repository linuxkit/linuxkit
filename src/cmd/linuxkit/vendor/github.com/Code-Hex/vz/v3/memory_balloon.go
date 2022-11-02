package vz

/*
#cgo darwin CFLAGS: -mmacosx-version-min=11 -x objective-c -fno-objc-arc
#cgo darwin LDFLAGS: -lobjc -framework Foundation -framework Virtualization
# include "virtualization_11.h"
*/
import "C"
import (
	"github.com/Code-Hex/vz/v3/internal/objc"
)

// MemoryBalloonDeviceConfiguration for a memory balloon device configuration.
type MemoryBalloonDeviceConfiguration interface {
	objc.NSObject

	memoryBalloonDeviceConfiguration()
}

type baseMemoryBalloonDeviceConfiguration struct{}

func (*baseMemoryBalloonDeviceConfiguration) memoryBalloonDeviceConfiguration() {}

var _ MemoryBalloonDeviceConfiguration = (*VirtioTraditionalMemoryBalloonDeviceConfiguration)(nil)

// VirtioTraditionalMemoryBalloonDeviceConfiguration is a configuration of the Virtio traditional memory balloon device.
//
// see: https://developer.apple.com/documentation/virtualization/vzvirtiotraditionalmemoryballoondeviceconfiguration?language=objc
type VirtioTraditionalMemoryBalloonDeviceConfiguration struct {
	*pointer

	*baseMemoryBalloonDeviceConfiguration
}

// NewVirtioTraditionalMemoryBalloonDeviceConfiguration creates a new VirtioTraditionalMemoryBalloonDeviceConfiguration.
//
// This is only supported on macOS 11 and newer, error will
// be returned on older versions.
func NewVirtioTraditionalMemoryBalloonDeviceConfiguration() (*VirtioTraditionalMemoryBalloonDeviceConfiguration, error) {
	if err := macOSAvailable(11); err != nil {
		return nil, err
	}

	config := &VirtioTraditionalMemoryBalloonDeviceConfiguration{
		pointer: objc.NewPointer(
			C.newVZVirtioTraditionalMemoryBalloonDeviceConfiguration(),
		),
	}
	objc.SetFinalizer(config, func(self *VirtioTraditionalMemoryBalloonDeviceConfiguration) {
		objc.Release(self)
	})
	return config, nil
}
