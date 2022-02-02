package vz

/*
#cgo darwin CFLAGS: -x objective-c -fno-objc-arc
#cgo darwin LDFLAGS: -lobjc -framework Foundation -framework Virtualization
# include "virtualization.h"
*/
import "C"
import "runtime"

type baseStorageDeviceAttachment struct{}

func (*baseStorageDeviceAttachment) storageDeviceAttachment() {}

// StorageDeviceAttachment for a storage device attachment.
//
// A storage device attachment defines how a virtual machine storage device interfaces with the host system.
// see: https://developer.apple.com/documentation/virtualization/vzstoragedeviceattachment?language=objc
type StorageDeviceAttachment interface {
	NSObject

	storageDeviceAttachment()
}

var _ StorageDeviceAttachment = (*DiskImageStorageDeviceAttachment)(nil)

// DiskImageStorageDeviceAttachment is a storage device attachment using a disk image to implement the storage.
//
// This storage device attachment uses a disk image on the host file system as the drive of the storage device.
// Only raw data disk images are supported.
// see: https://developer.apple.com/documentation/virtualization/vzdiskimagestoragedeviceattachment?language=objc
type DiskImageStorageDeviceAttachment struct {
	pointer

	*baseStorageDeviceAttachment
}

// NewDiskImageStorageDeviceAttachment initialize the attachment from a local file path.
// Returns error is not nil, assigned with the error if the initialization failed.
//
// - diskPath is local file URL to the disk image in RAW format.
// - readOnly if YES, the device attachment is read-only, otherwise the device can write data to the disk image.
func NewDiskImageStorageDeviceAttachment(diskPath string, readOnly bool) (*DiskImageStorageDeviceAttachment, error) {
	nserr := newNSErrorAsNil()
	nserrPtr := nserr.Ptr()

	diskPathChar := charWithGoString(diskPath)
	defer diskPathChar.Free()
	attachment := &DiskImageStorageDeviceAttachment{
		pointer: pointer{
			ptr: C.newVZDiskImageStorageDeviceAttachment(
				diskPathChar.CString(),
				C.bool(readOnly),
				&nserrPtr,
			),
		},
	}
	if err := newNSError(nserrPtr); err != nil {
		return nil, err
	}
	runtime.SetFinalizer(attachment, func(self *DiskImageStorageDeviceAttachment) {
		self.Release()
	})
	return attachment, nil
}

// StorageDeviceConfiguration for a storage device configuration.
type StorageDeviceConfiguration interface {
	NSObject

	storageDeviceConfiguration()
}

type baseStorageDeviceConfiguration struct{}

func (*baseStorageDeviceConfiguration) storageDeviceConfiguration() {}

var _ StorageDeviceConfiguration = (*VirtioBlockDeviceConfiguration)(nil)

// VirtioBlockDeviceConfiguration is a configuration of a paravirtualized storage device of type Virtio Block Device.
//
// This device configuration creates a storage device using paravirtualization.
// The emulated device follows the Virtio Block Device specification.
//
// The host implementation of the device is done through an attachment subclassing VZStorageDeviceAttachment
// like VZDiskImageStorageDeviceAttachment.
// see: https://developer.apple.com/documentation/virtualization/vzvirtioblockdeviceconfiguration?language=objc
type VirtioBlockDeviceConfiguration struct {
	pointer

	*baseStorageDeviceConfiguration
}

// NewVirtioBlockDeviceConfiguration initialize a VZVirtioBlockDeviceConfiguration with a device attachment.
//
// - attachment The storage device attachment. This defines how the virtualized device operates on the host side.
func NewVirtioBlockDeviceConfiguration(attachment StorageDeviceAttachment) *VirtioBlockDeviceConfiguration {
	config := &VirtioBlockDeviceConfiguration{
		pointer: pointer{
			ptr: C.newVZVirtioBlockDeviceConfiguration(
				attachment.Ptr(),
			),
		},
	}
	runtime.SetFinalizer(config, func(self *VirtioBlockDeviceConfiguration) {
		self.Release()
	})
	return config
}
