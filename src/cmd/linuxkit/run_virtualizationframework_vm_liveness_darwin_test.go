//go:build darwin && cgo
// +build darwin,cgo

package main

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

type virtualizationFrameworkVMLivenessProbe struct {
	finalized chan struct{}
}

func newVirtualizationFrameworkVMLivenessProbe(finalized chan struct{}) *virtualizationFrameworkVMLivenessProbe {
	probe := &virtualizationFrameworkVMLivenessProbe{finalized: finalized}
	runtime.SetFinalizer(probe, func(*virtualizationFrameworkVMLivenessProbe) {
		close(finalized)
	})
	return probe
}

func TestVirtualizationFrameworkLifecycleRetainsVMUntilReturn(t *testing.T) {
	finalized := make(chan struct{})
	started := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan error, 1)
	workerDone := make(chan struct{})
	var releaseOnce sync.Once
	releaseLifecycle := func() {
		releaseOnce.Do(func() { close(release) })
	}

	go func() {
		defer close(workerDone)
		finished <- withVirtualizationFrameworkVMLifecycleAlive(
			newVirtualizationFrameworkVMLivenessProbe(finalized),
			func() error {
				close(started)
				<-release
				return nil
			},
		)
	}()
	t.Cleanup(func() {
		releaseLifecycle()
		<-workerDone
	})
	<-started

	for range 3 {
		runtime.GC()
		select {
		case <-finalized:
			t.Fatal("VM was finalized while lifecycle was still running")
		case <-time.After(10 * time.Millisecond):
		}
	}

	releaseLifecycle()
	if err := <-finished; err != nil {
		t.Fatalf("lifecycle failed: %v", err)
	}

	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	for {
		runtime.GC()
		select {
		case <-finalized:
			return
		case <-time.After(10 * time.Millisecond):
		case <-deadline.C:
			t.Fatal("VM finalizer did not run after lifecycle returned")
		}
	}
}
