//go:build darwin && cgo
// +build darwin,cgo

package main

import "runtime"

// Keep the VM reachable until its lifecycle callback returns. The vz wrapper
// owns native state observers through a finalizer, so a live lifecycle also
// requires the Go VM value to remain live.
func withVirtualizationFrameworkVMLifecycleAlive[VM any](vm VM, lifecycle func() error) error {
	defer runtime.KeepAlive(vm)
	return lifecycle()
}
