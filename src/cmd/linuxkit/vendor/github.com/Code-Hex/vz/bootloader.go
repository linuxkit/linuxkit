package vz

/*
#cgo darwin CFLAGS: -x objective-c -fno-objc-arc
#cgo darwin LDFLAGS: -lobjc -framework Foundation -framework Virtualization
# include "virtualization.h"
*/
import "C"
import (
	"fmt"
	"runtime"
)

// BootLoader is the interface of boot loader definitions.
// see: LinuxBootLoader
type BootLoader interface {
	NSObject

	bootLoader()
}

type baseBootLoader struct{}

func (*baseBootLoader) bootLoader() {}

var _ BootLoader = (*LinuxBootLoader)(nil)

// LinuxBootLoader Boot loader configuration for a Linux kernel.
type LinuxBootLoader struct {
	vmlinuzPath string
	initrdPath  string
	cmdLine     string
	pointer

	*baseBootLoader
}

func (b *LinuxBootLoader) String() string {
	return fmt.Sprintf(
		"vmlinuz: %q, initrd: %q, command-line: %q",
		b.vmlinuzPath,
		b.initrdPath,
		b.cmdLine,
	)
}

type LinuxBootLoaderOption func(b *LinuxBootLoader)

// WithCommandLine sets the command-line parameters.
// see: https://www.kernel.org/doc/html/latest/admin-guide/kernel-parameters.html
func WithCommandLine(cmdLine string) LinuxBootLoaderOption {
	return func(b *LinuxBootLoader) {
		b.cmdLine = cmdLine
		cs := charWithGoString(cmdLine)
		defer cs.Free()
		C.setCommandLineVZLinuxBootLoader(b.Ptr(), cs.CString())
	}
}

// WithInitrd sets the optional initial RAM disk.
func WithInitrd(initrdPath string) LinuxBootLoaderOption {
	return func(b *LinuxBootLoader) {
		b.initrdPath = initrdPath
		cs := charWithGoString(initrdPath)
		defer cs.Free()
		C.setInitialRamdiskURLVZLinuxBootLoader(b.Ptr(), cs.CString())
	}
}

// NewLinuxBootLoader creates a LinuxBootLoader with the Linux kernel passed as Path.
func NewLinuxBootLoader(vmlinuz string, opts ...LinuxBootLoaderOption) *LinuxBootLoader {
	vmlinuzPath := charWithGoString(vmlinuz)
	defer vmlinuzPath.Free()
	bootLoader := &LinuxBootLoader{
		vmlinuzPath: vmlinuz,
		pointer: pointer{
			ptr: C.newVZLinuxBootLoader(
				vmlinuzPath.CString(),
			),
		},
	}
	runtime.SetFinalizer(bootLoader, func(self *LinuxBootLoader) {
		self.Release()
	})
	for _, opt := range opts {
		opt(bootLoader)
	}
	return bootLoader
}
