package vz

/*
#cgo darwin CFLAGS: -mmacosx-version-min=11 -x objective-c -fno-objc-arc
#cgo darwin LDFLAGS: -lobjc -framework Foundation -framework Virtualization
# include "virtualization_11.h"
# include "virtualization_12_3.h"
# include "virtualization_13.h"
*/
import "C"
import (
	"os"

	"github.com/Code-Hex/vz/v3/internal/objc"
)

type baseStorageDeviceAttachment struct{}

func (*baseStorageDeviceAttachment) storageDeviceAttachment() {}

// StorageDeviceAttachment for a storage device attachment.
//
// A storage device attachment defines how a virtual machine storage device interfaces with the host system.
// see: https://developer.apple.com/documentation/virtualization/vzstoragedeviceattachment?language=objc
type StorageDeviceAttachment interface {
	objc.NSObject

	storageDeviceAttachment()
}

var _ StorageDeviceAttachment = (*DiskImageStorageDeviceAttachment)(nil)

// DiskImageStorageDeviceAttachment is a storage device attachment using a disk image to implement the storage.
//
// This storage device attachment uses a disk image on the host file system as the drive of the storage device.
// Only raw data disk images are supported.
// see: https://developer.apple.com/documentation/virtualization/vzdiskimagestoragedeviceattachment?language=objc
type DiskImageStorageDeviceAttachment struct {
	*pointer

	*baseStorageDeviceAttachment
}

// NewDiskImageStorageDeviceAttachment initialize the attachment from a local file path.
// Returns error is not nil, assigned with the error if the initialization failed.
//
// - diskPath is local file URL to the disk image in RAW format.
// - readOnly if YES, the device attachment is read-only, otherwise the device can write data to the disk image.
//
// This is only supported on macOS 11 and newer, error will
// be returned on older versions.
func NewDiskImageStorageDeviceAttachment(diskPath string, readOnly bool) (*DiskImageStorageDeviceAttachment, error) {
	if err := macOSAvailable(11); err != nil {
		return nil, err
	}
	if _, err := os.Stat(diskPath); err != nil {
		return nil, err
	}

	nserrPtr := newNSErrorAsNil()

	diskPathChar := charWithGoString(diskPath)
	defer diskPathChar.Free()
	attachment := &DiskImageStorageDeviceAttachment{
		pointer: objc.NewPointer(
			C.newVZDiskImageStorageDeviceAttachment(
				diskPathChar.CString(),
				C.bool(readOnly),
				&nserrPtr,
			),
		),
	}
	if err := newNSError(nserrPtr); err != nil {
		return nil, err
	}
	objc.SetFinalizer(attachment, func(self *DiskImageStorageDeviceAttachment) {
		objc.Release(self)
	})
	return attachment, nil
}

// StorageDeviceConfiguration for a storage device configuration.
type StorageDeviceConfiguration interface {
	objc.NSObject

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
	*pointer

	*baseStorageDeviceConfiguration

	blockDeviceIdentifier string
}

// NewVirtioBlockDeviceConfiguration initialize a VZVirtioBlockDeviceConfiguration with a device attachment.
//
// - attachment The storage device attachment. This defines how the virtualized device operates on the host side.
//
// This is only supported on macOS 11 and newer, error will
// be returned on older versions.
func NewVirtioBlockDeviceConfiguration(attachment StorageDeviceAttachment) (*VirtioBlockDeviceConfiguration, error) {
	if err := macOSAvailable(11); err != nil {
		return nil, err
	}

	config := &VirtioBlockDeviceConfiguration{
		pointer: objc.NewPointer(
			C.newVZVirtioBlockDeviceConfiguration(
				objc.Ptr(attachment),
			),
		),
	}
	objc.SetFinalizer(config, func(self *VirtioBlockDeviceConfiguration) {
		objc.Release(self)
	})
	return config, nil
}

// BlockDeviceIdentifier returns the device identifier is a string identifying the Virtio block device.
// Empty string by default.
//
// The identifier can be retrieved in the guest via a VIRTIO_BLK_T_GET_ID request.
//
// This is only supported on macOS 12.3 and newer, error will be returned on older versions.
//
// see: https://developer.apple.com/documentation/virtualization/vzvirtioblockdeviceconfiguration/3917717-blockdeviceidentifier
func (v *VirtioBlockDeviceConfiguration) BlockDeviceIdentifier() (string, error) {
	if err := macOSAvailable(12.3); err != nil {
		return "", err
	}
	return v.blockDeviceIdentifier, nil
}

// SetBlockDeviceIdentifier sets the device identifier is a string identifying the Virtio block device.
//
// The device identifier must be at most 20 bytes in length and ASCII-encodable.
//
// This is only supported on macOS 12.3 and newer, error will be returned on older versions.
//
// see: https://developer.apple.com/documentation/virtualization/vzvirtioblockdeviceconfiguration/3917717-blockdeviceidentifier
func (v *VirtioBlockDeviceConfiguration) SetBlockDeviceIdentifier(identifier string) error {
	if err := macOSAvailable(12.3); err != nil {
		return err
	}
	idChar := charWithGoString(identifier)
	defer idChar.Free()

	nserrPtr := newNSErrorAsNil()
	C.setBlockDeviceIdentifierVZVirtioBlockDeviceConfiguration(
		objc.Ptr(v),
		idChar.CString(),
		&nserrPtr,
	)
	if err := newNSError(nserrPtr); err != nil {
		return err
	}
	v.blockDeviceIdentifier = identifier
	return nil
}

// USBMassStorageDeviceConfiguration is a configuration of a USB Mass Storage storage device.
//
// This device configuration creates a storage device that conforms to the USB Mass Storage specification.
//
// see: https://developer.apple.com/documentation/virtualization/vzusbmassstoragedeviceconfiguration?language=objc
type USBMassStorageDeviceConfiguration struct {
	*pointer

	*baseStorageDeviceConfiguration

	// marking as currently reachable.
	// This ensures that the object is not freed, and its finalizer is not run
	attachment StorageDeviceAttachment
}

// NewUSBMassStorageDeviceConfiguration initialize a USBMassStorageDeviceConfiguration
// with a device attachment.
//
// This is only supported on macOS 13 and newer, error will
// be returned on older versions.
func NewUSBMassStorageDeviceConfiguration(attachment StorageDeviceAttachment) (*USBMassStorageDeviceConfiguration, error) {
	if err := macOSAvailable(13); err != nil {
		return nil, err
	}
	usbMass := &USBMassStorageDeviceConfiguration{
		pointer: objc.NewPointer(
			C.newVZUSBMassStorageDeviceConfiguration(objc.Ptr(attachment)),
		),
		attachment: attachment,
	}
	objc.SetFinalizer(usbMass, func(self *USBMassStorageDeviceConfiguration) {
		objc.Release(self)
	})
	return usbMass, nil
}
