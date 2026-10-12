package main

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type lifecycleTestState int

const (
	lifecycleTestStopped lifecycleTestState = iota
	lifecycleTestRunning
	lifecycleTestError
)

type fakeVirtualizationFrameworkVM struct {
	start       func() error
	requestStop func() (bool, error)
	stop        func() error
	startCalls  atomic.Int32
	requestCall atomic.Int32
	stopCalls   atomic.Int32
}

func (vm *fakeVirtualizationFrameworkVM) Start() error {
	vm.startCalls.Add(1)
	if vm.start != nil {
		return vm.start()
	}
	return nil
}

func (vm *fakeVirtualizationFrameworkVM) RequestStop() (bool, error) {
	vm.requestCall.Add(1)
	if vm.requestStop != nil {
		return vm.requestStop()
	}
	return true, nil
}

func (vm *fakeVirtualizationFrameworkVM) Stop() error {
	vm.stopCalls.Add(1)
	if vm.stop != nil {
		return vm.stop()
	}
	return nil
}

func TestVirtualizationFrameworkLifecycleNaturalStop(t *testing.T) {
	states := make(chan lifecycleTestState, 1)
	states <- lifecycleTestStopped
	vm := &fakeVirtualizationFrameworkVM{}

	err := runVirtualizationFrameworkVMLifecycle(
		context.Background(), nil, vm, states,
		lifecycleTestStopped, lifecycleTestRunning, lifecycleTestError,
		shortLifecycleTimeouts(), virtualizationFrameworkLifecycleHooks{},
	)
	if err != nil {
		t.Fatalf("runVirtualizationFrameworkVMLifecycle() error = %v, want nil", err)
	}
	if got := vm.requestCall.Load(); got != 0 {
		t.Fatalf("RequestStop calls = %d, want 0", got)
	}
	if got := vm.stopCalls.Load(); got != 0 {
		t.Fatalf("Stop calls = %d, want 0", got)
	}
}

func TestVirtualizationFrameworkLifecycleAcceptedRequestStopsGracefully(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	states := make(chan lifecycleTestState, 1)
	vm := &fakeVirtualizationFrameworkVM{
		start: func() error {
			cancel()
			return nil
		},
		requestStop: func() (bool, error) {
			states <- lifecycleTestStopped
			return true, nil
		},
	}
	err := runTestLifecycle(ctx, nil, vm, states, shortLifecycleTimeouts())
	if err != nil {
		t.Fatalf("runTestLifecycle() error = %v, want nil", err)
	}
	if got := vm.stopCalls.Load(); got != 0 {
		t.Fatalf("Stop calls = %d, want no force stop after graceful shutdown", got)
	}
}

func TestVirtualizationFrameworkLifecycleClosedAndErrorStatesFail(t *testing.T) {
	tests := []struct {
		name     string
		states   func() <-chan lifecycleTestState
		wantText string
	}{
		{
			name: "closed state stream",
			states: func() <-chan lifecycleTestState {
				states := make(chan lifecycleTestState)
				close(states)
				return states
			},
			wantText: "state notification channel closed",
		},
		{
			name: "error state",
			states: func() <-chan lifecycleTestState {
				states := make(chan lifecycleTestState, 1)
				states <- lifecycleTestError
				return states
			},
			wantText: "entered error state",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm := &fakeVirtualizationFrameworkVM{
				requestStop: func() (bool, error) { return false, nil },
			}
			err := runTestLifecycle(context.Background(), nil, vm, tt.states(), shortLifecycleTimeouts())
			if err == nil || !strings.Contains(err.Error(), tt.wantText) {
				t.Fatalf("runTestLifecycle() error = %v, want text %q", err, tt.wantText)
			}
			if got := vm.stopCalls.Load(); got != 1 {
				t.Fatalf("Stop calls = %d, want 1", got)
			}
		})
	}
}

func TestVirtualizationFrameworkLifecycleRequestStopOutcomes(t *testing.T) {
	requestErr := errors.New("request failed")
	tests := []struct {
		name       string
		accepted   bool
		requestErr error
		wantErr    bool
	}{
		{name: "not accepted"},
		{name: "request error", requestErr: requestErr, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			vm := &fakeVirtualizationFrameworkVM{
				start: func() error {
					cancel()
					return nil
				},
				requestStop: func() (bool, error) { return tt.accepted, tt.requestErr },
			}
			states := make(chan lifecycleTestState)
			err := runTestLifecycle(ctx, nil, vm, states, shortLifecycleTimeouts())
			if tt.wantErr && !errors.Is(err, requestErr) {
				t.Fatalf("runTestLifecycle() error = %v, want wrapped %v", err, requestErr)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("runTestLifecycle() error = %v, want nil", err)
			}
			if got := vm.stopCalls.Load(); got != 1 {
				t.Fatalf("Stop calls = %d, want 1", got)
			}
		})
	}
}

func TestVirtualizationFrameworkLifecycleAcceptedRequestFallsBackAfterGrace(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	vm := &fakeVirtualizationFrameworkVM{
		start: func() error {
			cancel()
			return nil
		},
		requestStop: func() (bool, error) { return true, nil },
	}
	started := time.Now()
	err := runTestLifecycle(ctx, nil, vm, make(chan lifecycleTestState), virtualizationFrameworkLifecycleTimeouts{
		GracefulShutdown: 20 * time.Millisecond,
		ForceStop:        40 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("runTestLifecycle() error = %v, want nil", err)
	}
	if elapsed := time.Since(started); elapsed > 200*time.Millisecond {
		t.Fatalf("shutdown took %s, want it bounded by the injected short timeout", elapsed)
	}
	if got := vm.stopCalls.Load(); got != 1 {
		t.Fatalf("Stop calls = %d, want 1", got)
	}
}

func TestVirtualizationFrameworkLifecycleRequestTimeoutFallsBack(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})
	vm := &fakeVirtualizationFrameworkVM{
		start: func() error {
			cancel()
			return nil
		},
		requestStop: func() (bool, error) {
			close(requestStarted)
			<-releaseRequest
			return true, nil
		},
	}
	result := make(chan error, 1)
	go func() {
		result <- runTestLifecycle(ctx, nil, vm, make(chan lifecycleTestState), virtualizationFrameworkLifecycleTimeouts{
			GracefulShutdown: 20 * time.Millisecond,
			ForceStop:        40 * time.Millisecond,
		})
	}()
	<-requestStarted
	err := waitLifecycleResult(t, result)
	close(releaseRequest)
	if err == nil || !strings.Contains(err.Error(), "stop request timed out") {
		t.Fatalf("runTestLifecycle() error = %v, want bounded stop-request timeout error", err)
	}
	if got := vm.stopCalls.Load(); got != 1 {
		t.Fatalf("Stop calls = %d, want 1", got)
	}
}

func TestVirtualizationFrameworkLifecycleSecondSignalForcesStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	forceStop := make(chan struct{})
	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})
	vm := &fakeVirtualizationFrameworkVM{
		start: func() error {
			cancel()
			return nil
		},
		requestStop: func() (bool, error) {
			close(requestStarted)
			<-releaseRequest
			return true, nil
		},
	}
	result := make(chan error, 1)
	go func() {
		result <- runTestLifecycle(ctx, forceStop, vm, make(chan lifecycleTestState), shortLifecycleTimeouts())
	}()
	<-requestStarted
	close(forceStop)
	if err := waitLifecycleResult(t, result); err != nil {
		t.Fatalf("runTestLifecycle() error = %v, want nil", err)
	}
	close(releaseRequest)
	if got := vm.stopCalls.Load(); got != 1 {
		t.Fatalf("Stop calls = %d, want 1", got)
	}
}

func TestVirtualizationFrameworkLifecycleStopErrorAndTimeout(t *testing.T) {
	stopErr := errors.New("force stop failed")
	tests := []struct {
		name    string
		stopErr error
		block   bool
		want    string
	}{
		{name: "stop error", stopErr: stopErr, want: "force stop failed"},
		{name: "stop timeout", block: true, want: "timed out"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			releaseStop := make(chan struct{})
			vm := &fakeVirtualizationFrameworkVM{
				start: func() error {
					cancel()
					return nil
				},
				requestStop: func() (bool, error) { return false, nil },
				stop: func() error {
					if tt.block {
						<-releaseStop
					}
					return tt.stopErr
				},
			}
			timeouts := virtualizationFrameworkLifecycleTimeouts{GracefulShutdown: 20 * time.Millisecond, ForceStop: 20 * time.Millisecond}
			err := runTestLifecycle(ctx, nil, vm, make(chan lifecycleTestState), timeouts)
			close(releaseStop)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("runTestLifecycle() error = %v, want text %q", err, tt.want)
			}
		})
	}
}

func TestVirtualizationFrameworkLifecycleStartFailureAndPreStartCancellation(t *testing.T) {
	startErr := errors.New("start failed")
	vm := &fakeVirtualizationFrameworkVM{start: func() error { return startErr }}
	err := runTestLifecycle(context.Background(), nil, vm, make(chan lifecycleTestState), shortLifecycleTimeouts())
	if !errors.Is(err, startErr) {
		t.Fatalf("runTestLifecycle() error = %v, want wrapped %v", err, startErr)
	}
	if got := vm.stopCalls.Load(); got != 0 {
		t.Fatalf("Stop calls after start failure = %d, want 0", got)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	vm = &fakeVirtualizationFrameworkVM{}
	err = runTestLifecycle(ctx, nil, vm, make(chan lifecycleTestState), shortLifecycleTimeouts())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("runTestLifecycle() error = %v, want context.Canceled", err)
	}
	if got := vm.startCalls.Load(); got != 0 {
		t.Fatalf("Start calls after pre-start cancellation = %d, want 0", got)
	}
}

func TestVirtualizationFrameworkLifecycleCancellationDuringStartRemainsOwned(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	startEntered := make(chan struct{})
	releaseStart := make(chan struct{})
	states := make(chan lifecycleTestState, 1)
	vm := &fakeVirtualizationFrameworkVM{
		start: func() error {
			close(startEntered)
			<-releaseStart
			return nil
		},
		requestStop: func() (bool, error) {
			states <- lifecycleTestStopped
			return true, nil
		},
	}
	result := make(chan error, 1)
	go func() { result <- runTestLifecycle(ctx, nil, vm, states, shortLifecycleTimeouts()) }()
	<-startEntered
	cancel()
	select {
	case <-result:
		t.Fatal("lifecycle returned while Start was still running; VM ownership would be lost")
	default:
	}
	if got := vm.requestCall.Load(); got != 0 {
		t.Fatalf("RequestStop calls before Start completed = %d, want 0", got)
	}
	close(releaseStart)
	if err := waitLifecycleResult(t, result); err != nil {
		t.Fatalf("runTestLifecycle() error = %v, want nil", err)
	}
	if got := vm.requestCall.Load(); got != 1 {
		t.Fatalf("RequestStop calls after Start completed = %d, want 1", got)
	}
}

func TestVirtualizationFrameworkLifecycleNaturalStopWhileRequestBlocks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})
	states := make(chan lifecycleTestState)
	vm := &fakeVirtualizationFrameworkVM{
		start: func() error {
			cancel()
			return nil
		},
		requestStop: func() (bool, error) {
			close(requestStarted)
			<-releaseRequest
			return true, nil
		},
	}
	result := make(chan error, 1)
	go func() { result <- runTestLifecycle(ctx, nil, vm, states, shortLifecycleTimeouts()) }()
	<-requestStarted
	started := time.Now()
	states <- lifecycleTestStopped
	if err := waitLifecycleResult(t, result); err != nil {
		t.Fatalf("runTestLifecycle() error = %v, want nil", err)
	}
	if elapsed := time.Since(started); elapsed > 100*time.Millisecond {
		t.Fatalf("natural stop waited %s for blocked RequestStop", elapsed)
	}
	close(releaseRequest)
}

func TestVirtualizationFrameworkLifecycleStoppedStateConfirmsStopError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stopStarted := make(chan struct{})
	releaseStop := make(chan struct{})
	states := make(chan lifecycleTestState)
	vm := &fakeVirtualizationFrameworkVM{
		start: func() error {
			cancel()
			return nil
		},
		requestStop: func() (bool, error) { return false, nil },
		stop: func() error {
			close(stopStarted)
			<-releaseStop
			return errors.New("VM already stopped")
		},
	}
	result := make(chan error, 1)
	go func() { result <- runTestLifecycle(ctx, nil, vm, states, shortLifecycleTimeouts()) }()
	<-stopStarted
	states <- lifecycleTestStopped
	err := waitLifecycleResult(t, result)
	close(releaseStop)
	if err != nil {
		t.Fatalf("runTestLifecycle() error = %v, want stopped-state confirmation to resolve force-stop race", err)
	}
}

func runTestLifecycle(
	ctx context.Context,
	forceStop <-chan struct{},
	vm virtualizationFrameworkVMOperations,
	states <-chan lifecycleTestState,
	timeouts virtualizationFrameworkLifecycleTimeouts,
) error {
	return runVirtualizationFrameworkVMLifecycle(
		ctx,
		forceStop,
		vm,
		states,
		lifecycleTestStopped,
		lifecycleTestRunning,
		lifecycleTestError,
		timeouts,
		virtualizationFrameworkLifecycleHooks{},
	)
}

func shortLifecycleTimeouts() virtualizationFrameworkLifecycleTimeouts {
	return virtualizationFrameworkLifecycleTimeouts{
		GracefulShutdown: 40 * time.Millisecond,
		ForceStop:        40 * time.Millisecond,
	}
}

func waitLifecycleResult(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(time.Second):
		t.Fatal("lifecycle did not return within one second")
		return nil
	}
}
