package vz

/*
#cgo darwin CFLAGS: -x objective-c -fno-objc-arc
#cgo darwin LDFLAGS: -lobjc -framework Foundation -framework Virtualization
# include "virtualization.h"
*/
import "C"
import (
	"os"
	"runtime"
)

// SerialPortAttachment interface for a serial port attachment.
//
// A serial port attachment defines how the virtual machine's serial port interfaces with the host system.
type SerialPortAttachment interface {
	NSObject

	serialPortAttachment()
}

type baseSerialPortAttachment struct{}

func (*baseSerialPortAttachment) serialPortAttachment() {}

var _ SerialPortAttachment = (*FileHandleSerialPortAttachment)(nil)

// FileHandleSerialPortAttachment defines a serial port attachment from a file handle.
//
// Data written to fileHandleForReading goes to the guest. Data sent from the guest appears on fileHandleForWriting.
// see: https://developer.apple.com/documentation/virtualization/vzfilehandleserialportattachment?language=objc
type FileHandleSerialPortAttachment struct {
	pointer

	*baseSerialPortAttachment
}

// NewFileHandleSerialPortAttachment intialize the FileHandleSerialPortAttachment from file handles.
//
// read parameter is an *os.File for reading from the file.
// write parameter is an *os.File for writing to the file.
func NewFileHandleSerialPortAttachment(read, write *os.File) *FileHandleSerialPortAttachment {
	attachment := &FileHandleSerialPortAttachment{
		pointer: pointer{
			ptr: C.newVZFileHandleSerialPortAttachment(
				C.int(read.Fd()),
				C.int(write.Fd()),
			),
		},
	}
	runtime.SetFinalizer(attachment, func(self *FileHandleSerialPortAttachment) {
		self.Release()
	})
	return attachment
}

var _ SerialPortAttachment = (*FileSerialPortAttachment)(nil)

// FileSerialPortAttachment defines a serial port attachment from a file.
//
// Any data sent by the guest on the serial interface is written to the file.
// No data is sent to the guest over serial with this attachment.
// see: https://developer.apple.com/documentation/virtualization/vzfileserialportattachment?language=objc
type FileSerialPortAttachment struct {
	pointer

	*baseSerialPortAttachment
}

// NewFileSerialPortAttachment initialize the FileSerialPortAttachment from a path of a file.
// If error is not nil, used to report errors if intialization fails.
//
// - path of the file for the attachment on the local file system.
// - shouldAppend True if the file should be opened in append mode, false otherwise.
//    When a file is opened in append mode, writing to that file will append to the end of it.
func NewFileSerialPortAttachment(path string, shouldAppend bool) (*FileSerialPortAttachment, error) {
	cpath := charWithGoString(path)
	defer cpath.Free()

	nserr := newNSErrorAsNil()
	nserrPtr := nserr.Ptr()
	attachment := &FileSerialPortAttachment{
		pointer: pointer{
			ptr: C.newVZFileSerialPortAttachment(
				cpath.CString(),
				C.bool(shouldAppend),
				&nserrPtr,
			),
		},
	}
	if err := newNSError(nserrPtr); err != nil {
		return nil, err
	}
	runtime.SetFinalizer(attachment, func(self *FileSerialPortAttachment) {
		self.Release()
	})
	return attachment, nil
}

// VirtioConsoleDeviceSerialPortConfiguration represents Virtio Console Serial Port Device.
//
// The device creates a console which enables communication between the host and the guest through the Virtio interface.
// The device sets up a single port on the Virtio console device.
// see: https://developer.apple.com/documentation/virtualization/vzvirtioconsoledeviceserialportconfiguration?language=objc
type VirtioConsoleDeviceSerialPortConfiguration struct {
	pointer
}

// NewVirtioConsoleDeviceSerialPortConfiguration creates a new NewVirtioConsoleDeviceSerialPortConfiguration.
func NewVirtioConsoleDeviceSerialPortConfiguration(attachment SerialPortAttachment) *VirtioConsoleDeviceSerialPortConfiguration {
	config := &VirtioConsoleDeviceSerialPortConfiguration{
		pointer: pointer{
			ptr: C.newVZVirtioConsoleDeviceSerialPortConfiguration(
				attachment.Ptr(),
			),
		},
	}
	runtime.SetFinalizer(config, func(self *VirtioConsoleDeviceSerialPortConfiguration) {
		self.Release()
	})
	return config
}
