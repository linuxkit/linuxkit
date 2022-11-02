//go:build darwin && arm64
// +build darwin,arm64

package vz

/*
#cgo darwin CFLAGS: -mmacosx-version-min=11 -x objective-c -fno-objc-arc
#cgo darwin LDFLAGS: -lobjc -framework Foundation -framework Virtualization
# include "virtualization_13_arm64.h"
*/
import "C"
import (
	"runtime/cgo"
	"unsafe"

	"github.com/Code-Hex/vz/v3/internal/objc"
)

// LinuxRosettaAvailability represents an availability of Rosetta support for Linux binaries.
//
//go:generate go run ./cmd/addtags -tags=darwin,arm64 -file linuxrosettaavailability_string_arm64.go stringer -type=LinuxRosettaAvailability -output=linuxrosettaavailability_string_arm64.go
type LinuxRosettaAvailability int

const (
	// LinuxRosettaAvailabilityNotSupported Rosetta support for Linux binaries is not available on the host system.
	LinuxRosettaAvailabilityNotSupported LinuxRosettaAvailability = iota

	// LinuxRosettaAvailabilityNotInstalled Rosetta support for Linux binaries is not installed on the host system.
	LinuxRosettaAvailabilityNotInstalled

	// LinuxRosettaAvailabilityInstalled Rosetta support for Linux is installed on the host system.
	LinuxRosettaAvailabilityInstalled
)

//export linuxInstallRosettaWithCompletionHandler
func linuxInstallRosettaWithCompletionHandler(cgoHandlerPtr, errPtr unsafe.Pointer) {
	cgoHandler := *(*cgo.Handle)(cgoHandlerPtr)

	handler := cgoHandler.Value().(func(error))

	if err := newNSError(errPtr); err != nil {
		handler(err)
	} else {
		handler(nil)
	}
}

// LinuxRosettaDirectoryShare directory share to enable Rosetta support for Linux binaries.
// see: https://developer.apple.com/documentation/virtualization/vzlinuxrosettadirectoryshare?language=objc
type LinuxRosettaDirectoryShare struct {
	*pointer

	*baseDirectoryShare
}

var _ DirectoryShare = (*LinuxRosettaDirectoryShare)(nil)

// NewLinuxRosettaDirectoryShare creates a new Rosetta directory share if Rosetta support
// for Linux binaries is installed.
//
// This is only supported on macOS 13 and newer, error will
// be returned on older versions.
func NewLinuxRosettaDirectoryShare() (*LinuxRosettaDirectoryShare, error) {
	if err := macOSAvailable(13); err != nil {
		return nil, err
	}
	nserrPtr := newNSErrorAsNil()
	ds := &LinuxRosettaDirectoryShare{
		pointer: objc.NewPointer(
			C.newVZLinuxRosettaDirectoryShare(&nserrPtr),
		),
	}
	if err := newNSError(nserrPtr); err != nil {
		return nil, err
	}
	objc.SetFinalizer(ds, func(self *LinuxRosettaDirectoryShare) {
		objc.Release(self)
	})
	return ds, nil
}

// LinuxRosettaDirectoryShareInstallRosetta download and install Rosetta support
// for Linux binaries if necessary.
//
// This is only supported on macOS 13 and newer, error will
// be returned on older versions.
func LinuxRosettaDirectoryShareInstallRosetta() error {
	if err := macOSAvailable(13); err != nil {
		return err
	}
	errCh := make(chan error, 1)
	cgoHandler := cgo.NewHandle(func(err error) {
		errCh <- err
	})
	C.linuxInstallRosetta(unsafe.Pointer(&cgoHandler))
	return <-errCh
}

// LinuxRosettaDirectoryShareAvailability checks the availability of Rosetta support
// for the directory share.
//
// This is only supported on macOS 13 and newer, LinuxRosettaAvailabilityNotSupported will
// be returned on older versions.
func LinuxRosettaDirectoryShareAvailability() LinuxRosettaAvailability {
	if err := macOSAvailable(13); err != nil {
		return LinuxRosettaAvailabilityNotSupported
	}
	return LinuxRosettaAvailability(C.availabilityVZLinuxRosettaDirectoryShare())
}
