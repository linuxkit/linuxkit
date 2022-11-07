package vz

/*
#cgo darwin CFLAGS: -mmacosx-version-min=11 -x objective-c -fno-objc-arc
#cgo darwin LDFLAGS: -lobjc -framework Foundation -framework Virtualization -framework Cocoa
# include "virtualization_11.h"
# include "virtualization_12.h"
# include "virtualization_13.h"
*/
import "C"
import (
	"runtime/cgo"
	"sync"
	"unsafe"

	"github.com/Code-Hex/vz/v3/internal/objc"
)

func init() {
	C.sharedApplication()
}

// VirtualMachineState represents execution state of the virtual machine.
//
//go:generate stringer -type=VirtualMachineState
type VirtualMachineState int

const (
	// VirtualMachineStateStopped Initial state before the virtual machine is started.
	VirtualMachineStateStopped VirtualMachineState = iota

	// VirtualMachineStateRunning Running virtual machine.
	VirtualMachineStateRunning

	// VirtualMachineStatePaused A started virtual machine is paused.
	// This state can only be transitioned from VirtualMachineStatePausing.
	VirtualMachineStatePaused

	// VirtualMachineStateError The virtual machine has encountered an internal error.
	VirtualMachineStateError

	// VirtualMachineStateStarting The virtual machine is configuring the hardware and starting.
	VirtualMachineStateStarting

	// VirtualMachineStatePausing The virtual machine is being paused.
	// This is the intermediate state between VirtualMachineStateRunning and VirtualMachineStatePaused.
	VirtualMachineStatePausing

	// VirtualMachineStateResuming The virtual machine is being resumed.
	// This is the intermediate state between VirtualMachineStatePaused and VirtualMachineStateRunning.
	VirtualMachineStateResuming

	// VZVirtualMachineStateStopping The virtual machine is being stopped.
	// This is the intermediate state between VZVirtualMachineStateRunning and VZVirtualMachineStateStop.
	VirtualMachineStateStopping
)

// VirtualMachine represents the entire state of a single virtual machine.
//
// A Virtual Machine is the emulation of a complete hardware machine of the same architecture as the real hardware machine.
// When executing the Virtual Machine, the Virtualization framework uses certain hardware resources and emulates others to provide isolation
// and great performance.
//
// The definition of a virtual machine starts with its configuration. This is done by setting up a VirtualMachineConfiguration struct.
// Once configured, the virtual machine can be started with (*VirtualMachine).Start() method.
//
// Creating a virtual machine using the Virtualization framework requires the app to have the "com.apple.security.virtualization" entitlement.
// see: https://developer.apple.com/documentation/virtualization/vzvirtualmachine?language=objc
type VirtualMachine struct {
	// id for this struct.
	id string

	// Indicate whether or not virtualization is available.
	//
	// If virtualization is unavailable, no VirtualMachineConfiguration will validate.
	// The validation error of the VirtualMachineConfiguration provides more information about why virtualization is unavailable.
	supported bool

	*pointer
	dispatchQueue unsafe.Pointer
	status        cgo.Handle

	mu sync.Mutex
}

type machineStatus struct {
	state       VirtualMachineState
	stateNotify chan VirtualMachineState

	mu sync.RWMutex
}

// NewVirtualMachine creates a new VirtualMachine with VirtualMachineConfiguration.
//
// The configuration must be valid. Validation can be performed at runtime with (*VirtualMachineConfiguration).Validate() method.
// The configuration is copied by the initializer.
//
// This is only supported on macOS 11 and newer, error will
// be returned on older versions.
func NewVirtualMachine(config *VirtualMachineConfiguration) (*VirtualMachine, error) {
	if err := macOSAvailable(11); err != nil {
		return nil, err
	}

	// should not call Free function for this string.
	cs := (*char)(objc.GetUUID())
	dispatchQueue := C.makeDispatchQueue(cs.CString())

	status := cgo.NewHandle(&machineStatus{
		state:       VirtualMachineState(0),
		stateNotify: make(chan VirtualMachineState),
	})

	v := &VirtualMachine{
		id: cs.String(),
		pointer: objc.NewPointer(
			C.newVZVirtualMachineWithDispatchQueue(
				objc.Ptr(config),
				dispatchQueue,
				unsafe.Pointer(&status),
			),
		),
		dispatchQueue: dispatchQueue,
		status:        status,
	}

	objc.SetFinalizer(v, func(self *VirtualMachine) {
		self.status.Delete()
		objc.ReleaseDispatch(self.dispatchQueue)
		objc.Release(self)
	})
	return v, nil
}

// SocketDevices return the list of socket devices configured on this virtual machine.
// Return an empty array if no socket device is configured.
//
// Since only NewVirtioSocketDeviceConfiguration is available in vz package,
// it will always return VirtioSocketDevice.
// see: https://developer.apple.com/documentation/virtualization/vzvirtualmachine/3656702-socketdevices?language=objc
func (v *VirtualMachine) SocketDevices() []*VirtioSocketDevice {
	nsArray := objc.NewNSArray(
		C.VZVirtualMachine_socketDevices(objc.Ptr(v)),
	)
	ptrs := nsArray.ToPointerSlice()
	socketDevices := make([]*VirtioSocketDevice, len(ptrs))
	for i, ptr := range ptrs {
		socketDevices[i] = newVirtioSocketDevice(ptr, v.dispatchQueue)
	}
	return socketDevices
}

//export changeStateOnObserver
func changeStateOnObserver(state C.int, cgoHandlerPtr unsafe.Pointer) {
	status := *(*cgo.Handle)(cgoHandlerPtr)
	// I expected it will not cause panic.
	// if caused panic, that's unexpected behavior.
	v, _ := status.Value().(*machineStatus)
	v.mu.Lock()
	newState := VirtualMachineState(state)
	v.state = newState
	// for non-blocking
	go func() { v.stateNotify <- newState }()
	v.mu.Unlock()
}

// State represents execution state of the virtual machine.
func (v *VirtualMachine) State() VirtualMachineState {
	// I expected it will not cause panic.
	// if caused panic, that's unexpected behavior.
	val, _ := v.status.Value().(*machineStatus)
	val.mu.RLock()
	defer val.mu.RUnlock()
	return val.state
}

// StateChangedNotify gets notification is changed execution state of the virtual machine.
func (v *VirtualMachine) StateChangedNotify() <-chan VirtualMachineState {
	// I expected it will not cause panic.
	// if caused panic, that's unexpected behavior.
	val, _ := v.status.Value().(*machineStatus)
	val.mu.RLock()
	defer val.mu.RUnlock()
	return val.stateNotify
}

// CanStart returns true if the machine is in a state that can be started.
func (v *VirtualMachine) CanStart() bool {
	return bool(C.vmCanStart(objc.Ptr(v), v.dispatchQueue))
}

// CanPause returns true if the machine is in a state that can be paused.
func (v *VirtualMachine) CanPause() bool {
	return bool(C.vmCanPause(objc.Ptr(v), v.dispatchQueue))
}

// CanResume returns true if the machine is in a state that can be resumed.
func (v *VirtualMachine) CanResume() bool {
	return (bool)(C.vmCanResume(objc.Ptr(v), v.dispatchQueue))
}

// CanRequestStop returns whether the machine is in a state where the guest can be asked to stop.
func (v *VirtualMachine) CanRequestStop() bool {
	return (bool)(C.vmCanRequestStop(objc.Ptr(v), v.dispatchQueue))
}

// CanStop returns whether the machine is in a state that can be stopped.
//
// This is only supported on macOS 12 and newer, false will always be returned
// on older versions.
func (v *VirtualMachine) CanStop() bool {
	if err := macOSAvailable(12); err != nil {
		return false
	}
	return (bool)(C.vmCanStop(objc.Ptr(v), v.dispatchQueue))
}

//export virtualMachineCompletionHandler
func virtualMachineCompletionHandler(cgoHandlerPtr, errPtr unsafe.Pointer) {
	cgoHandler := *(*cgo.Handle)(cgoHandlerPtr)

	handler := cgoHandler.Value().(func(error))

	if err := newNSError(errPtr); err != nil {
		handler(err)
	} else {
		handler(nil)
	}
}

func makeHandler() (func(error), chan error) {
	ch := make(chan error, 1)
	return func(err error) {
		ch <- err
		close(ch)
	}, ch
}

type virtualMachineStartOptions struct {
	macOSVirtualMachineStartOptionsPtr unsafe.Pointer
}

// VirtualMachineStartOption is an option for virtual machine start.
type VirtualMachineStartOption func(*virtualMachineStartOptions) error

// Start a virtual machine that is in either Stopped or Error state.
//
// If you want to listen status change events, use the "StateChangedNotify" method.
//
// If options are specified, also checks whether these options are
// available in use your macOS version available.
func (v *VirtualMachine) Start(opts ...VirtualMachineStartOption) error {
	o := &virtualMachineStartOptions{}
	for _, optFunc := range opts {
		if err := optFunc(o); err != nil {
			return err
		}
	}

	h, errCh := makeHandler()
	handler := cgo.NewHandle(h)
	defer handler.Delete()

	if o.macOSVirtualMachineStartOptionsPtr != nil {
		C.startWithOptionsCompletionHandler(
			objc.Ptr(v),
			v.dispatchQueue,
			o.macOSVirtualMachineStartOptionsPtr,
			unsafe.Pointer(&handler),
		)
	} else {
		C.startWithCompletionHandler(objc.Ptr(v), v.dispatchQueue, unsafe.Pointer(&handler))
	}
	return <-errCh
}

// Pause a virtual machine that is in Running state.
//
// If you want to listen status change events, use the "StateChangedNotify" method.
func (v *VirtualMachine) Pause() error {
	h, errCh := makeHandler()
	handler := cgo.NewHandle(h)
	defer handler.Delete()
	C.pauseWithCompletionHandler(objc.Ptr(v), v.dispatchQueue, unsafe.Pointer(&handler))
	return <-errCh
}

// Resume a virtual machine that is in the Paused state.
//
// If you want to listen status change events, use the "StateChangedNotify" method.
func (v *VirtualMachine) Resume() error {
	h, errCh := makeHandler()
	handler := cgo.NewHandle(h)
	defer handler.Delete()
	C.resumeWithCompletionHandler(objc.Ptr(v), v.dispatchQueue, unsafe.Pointer(&handler))
	return <-errCh
}

// RequestStop requests that the guest turns itself off.
//
// If returned error is not nil, assigned with the error if the request failed.
// Returns true if the request was made successfully.
func (v *VirtualMachine) RequestStop() (bool, error) {
	nserrPtr := newNSErrorAsNil()
	ret := (bool)(C.requestStopVirtualMachine(objc.Ptr(v), v.dispatchQueue, &nserrPtr))
	if err := newNSError(nserrPtr); err != nil {
		return ret, err
	}
	return ret, nil
}

// Stop stops a VM that’s in either a running or paused state.
//
// The completion handler returns an error object when the VM fails to stop,
// or nil if the stop was successful.
//
// If you want to listen status change events, use the "StateChangedNotify" method.
//
// Warning: This is a destructive operation. It stops the VM without
// giving the guest a chance to stop cleanly.
//
// This is only supported on macOS 12 and newer, error will be returned on older versions.
func (v *VirtualMachine) Stop() error {
	if err := macOSAvailable(12); err != nil {
		return err
	}
	h, errCh := makeHandler()
	handler := cgo.NewHandle(h)
	defer handler.Delete()
	C.stopWithCompletionHandler(objc.Ptr(v), v.dispatchQueue, unsafe.Pointer(&handler))
	return <-errCh
}

// StartGraphicApplication starts an application to display graphics of the VM.
//
// You must to call runtime.LockOSThread before calling this method.
//
// This is only supported on macOS 12 and newer, error will be returned on older versions.
func (v *VirtualMachine) StartGraphicApplication(width, height float64) error {
	if err := macOSAvailable(12); err != nil {
		return err
	}
	C.startVirtualMachineWindow(objc.Ptr(v), C.double(width), C.double(height))
	return nil
}
