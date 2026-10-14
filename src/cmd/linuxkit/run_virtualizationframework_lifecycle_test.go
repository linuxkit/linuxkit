package main

import (
	"context"
	"errors"
	"fmt"
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

func TestVirtualizationFrameworkLifecycleNaturalStopBeforeSignal(t *testing.T) {
	states := make(chan lifecycleTestState, 1)
	states <- lifecycleTestStopped
	vm := &fakeVirtualizationFrameworkVM{}

	err := runTestLifecycle(context.Background(), nil, vm, states, shortLifecycleTimeouts(), virtualizationFrameworkLifecycleHooks{})
	if err != nil {
		t.Fatalf("runTestLifecycle() error = %v, want nil", err)
	}
	if got := vm.requestCall.Load(); got != 0 {
		t.Fatalf("RequestStop calls = %d, want 0", got)
	}
	if got := vm.stopCalls.Load(); got != 0 {
		t.Fatalf("Stop calls = %d, want 0", got)
	}
}

func TestVirtualizationFrameworkLifecycleFirstSignalWaitsForNaturalStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	startReturned := make(chan struct{})
	requestStarted := make(chan struct{})
	states := make(chan lifecycleTestState, 1)
	logs := make(chan string, 16)
	vm := &fakeVirtualizationFrameworkVM{
		start: func() error {
			close(startReturned)
			return nil
		},
		requestStop: func() (bool, error) {
			close(requestStarted)
			return true, nil
		},
	}
	result := make(chan error, 1)
	go func() {
		result <- runTestLifecycle(ctx, nil, vm, states, shortLifecycleTimeouts(), lifecycleLogHooks(logs))
	}()
	waitLifecycleNotification(t, startReturned, "VM Start completion")
	cancel()
	waitLifecycleNotification(t, requestStarted, "graceful stop request")
	waitForLifecycleLog(t, logs, "guest accepted graceful shutdown request")
	assertLifecycleStillWaiting(t, result, vm)

	states <- lifecycleTestStopped
	if err := waitLifecycleResult(t, result); err != nil {
		t.Fatalf("runTestLifecycle() error = %v, want nil", err)
	}
	if got := vm.stopCalls.Load(); got != 0 {
		t.Fatalf("Stop calls = %d, want natural stop without force", got)
	}
}

func TestVirtualizationFrameworkLifecycleSecondSignalForcesStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	forceStop := make(chan struct{})
	startReturned := make(chan struct{})
	requestStarted := make(chan struct{})
	logs := make(chan string, 16)
	vm := &fakeVirtualizationFrameworkVM{
		start: func() error {
			close(startReturned)
			return nil
		},
		requestStop: func() (bool, error) {
			close(requestStarted)
			return true, nil
		},
	}
	result := make(chan error, 1)
	go func() {
		result <- runTestLifecycle(ctx, forceStop, vm, make(chan lifecycleTestState), shortLifecycleTimeouts(), lifecycleLogHooks(logs))
	}()
	waitLifecycleNotification(t, startReturned, "VM Start completion")
	cancel()
	waitLifecycleNotification(t, requestStarted, "graceful stop request")
	waitForLifecycleLog(t, logs, "guest accepted graceful shutdown request")
	assertLifecycleStillWaiting(t, result, vm)

	close(forceStop)
	if err := waitLifecycleResult(t, result); err != nil {
		t.Fatalf("runTestLifecycle() error = %v, want nil", err)
	}
	if got := vm.stopCalls.Load(); got != 1 {
		t.Fatalf("Stop calls = %d, want 1 after second signal", got)
	}
}

func TestVirtualizationFrameworkLifecycleDeclinedRequestWaitsForSecondSignal(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	forceStop := make(chan struct{})
	startReturned := make(chan struct{})
	requestStarted := make(chan struct{})
	logs := make(chan string, 16)
	vm := &fakeVirtualizationFrameworkVM{
		start: func() error {
			close(startReturned)
			return nil
		},
		requestStop: func() (bool, error) {
			close(requestStarted)
			return false, nil
		},
	}
	result := make(chan error, 1)
	go func() {
		result <- runTestLifecycle(ctx, forceStop, vm, make(chan lifecycleTestState), shortLifecycleTimeouts(), lifecycleLogHooks(logs))
	}()
	waitLifecycleNotification(t, startReturned, "VM Start completion")
	cancel()
	waitLifecycleNotification(t, requestStarted, "graceful stop request")
	waitForLifecycleLog(t, logs, "guest did not accept graceful shutdown request")
	assertLifecycleStillWaiting(t, result, vm)

	close(forceStop)
	if err := waitLifecycleResult(t, result); err != nil {
		t.Fatalf("runTestLifecycle() error = %v, want nil for a declined request followed by successful force stop", err)
	}
	if got := vm.stopCalls.Load(); got != 1 {
		t.Fatalf("Stop calls = %d, want 1 after second signal", got)
	}
}

func TestVirtualizationFrameworkLifecycleRequestErrorWaitsAndIsPreserved(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	forceStop := make(chan struct{})
	startReturned := make(chan struct{})
	requestStarted := make(chan struct{})
	requestErr := errors.New("request failed")
	logs := make(chan string, 16)
	vm := &fakeVirtualizationFrameworkVM{
		start: func() error {
			close(startReturned)
			return nil
		},
		requestStop: func() (bool, error) {
			close(requestStarted)
			return false, requestErr
		},
	}
	result := make(chan error, 1)
	go func() {
		result <- runTestLifecycle(ctx, forceStop, vm, make(chan lifecycleTestState), shortLifecycleTimeouts(), lifecycleLogHooks(logs))
	}()
	waitLifecycleNotification(t, startReturned, "VM Start completion")
	cancel()
	waitLifecycleNotification(t, requestStarted, "graceful stop request")
	waitForLifecycleLog(t, logs, "graceful shutdown request failed")
	assertLifecycleStillWaiting(t, result, vm)

	close(forceStop)
	if err := waitLifecycleResult(t, result); !errors.Is(err, requestErr) {
		t.Fatalf("runTestLifecycle() error = %v, want wrapped request error %v", err, requestErr)
	}
	if got := vm.stopCalls.Load(); got != 1 {
		t.Fatalf("Stop calls = %d, want 1 after second signal", got)
	}
}

func TestVirtualizationFrameworkLifecycleRequestErrorRemainsErrorAfterNaturalStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	startReturned := make(chan struct{})
	requestStarted := make(chan struct{})
	requestErr := errors.New("request failed")
	logs := make(chan string, 16)
	states := make(chan lifecycleTestState, 1)
	vm := &fakeVirtualizationFrameworkVM{
		start: func() error {
			close(startReturned)
			return nil
		},
		requestStop: func() (bool, error) {
			close(requestStarted)
			return false, requestErr
		},
	}
	result := make(chan error, 1)
	go func() {
		result <- runTestLifecycle(ctx, nil, vm, states, shortLifecycleTimeouts(), lifecycleLogHooks(logs))
	}()
	waitLifecycleNotification(t, startReturned, "VM Start completion")
	cancel()
	waitLifecycleNotification(t, requestStarted, "graceful stop request")
	waitForLifecycleLog(t, logs, "graceful shutdown request failed")
	assertLifecycleStillWaiting(t, result, vm)

	states <- lifecycleTestStopped
	if err := waitLifecycleResult(t, result); !errors.Is(err, requestErr) {
		t.Fatalf("runTestLifecycle() error = %v, want wrapped request error %v", err, requestErr)
	}
	if got := vm.stopCalls.Load(); got != 0 {
		t.Fatalf("Stop calls = %d, want natural stop without force", got)
	}
}

func TestVirtualizationFrameworkLifecycleSecondSignalDoesNotWaitForRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	forceStop := make(chan struct{})
	startReturned := make(chan struct{})
	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})
	requestFinished := make(chan struct{})
	vm := &fakeVirtualizationFrameworkVM{
		start: func() error {
			close(startReturned)
			return nil
		},
		requestStop: func() (bool, error) {
			close(requestStarted)
			<-releaseRequest
			close(requestFinished)
			return true, nil
		},
	}
	result := make(chan error, 1)
	go func() {
		result <- runTestLifecycle(ctx, forceStop, vm, make(chan lifecycleTestState), shortLifecycleTimeouts(), virtualizationFrameworkLifecycleHooks{})
	}()
	waitLifecycleNotification(t, startReturned, "VM Start completion")
	cancel()
	waitLifecycleNotification(t, requestStarted, "graceful stop request")
	close(forceStop)
	if err := waitLifecycleResult(t, result); err != nil {
		t.Fatalf("runTestLifecycle() error = %v, want force stop to proceed while RequestStop blocks", err)
	}
	if got := vm.stopCalls.Load(); got != 1 {
		t.Fatalf("Stop calls = %d, want 1", got)
	}
	close(releaseRequest)
	select {
	case <-requestFinished:
	case <-time.After(time.Second):
		t.Fatal("blocked RequestStop did not finish after release")
	}
}

func TestVirtualizationFrameworkLifecyclePreclosedSecondSignalSkipsRequest(t *testing.T) {
	forceStop := make(chan struct{})
	close(forceStop)
	vm := &fakeVirtualizationFrameworkVM{}
	err := runTestLifecycle(context.Background(), forceStop, vm, make(chan lifecycleTestState), shortLifecycleTimeouts(), virtualizationFrameworkLifecycleHooks{})
	if err != nil {
		t.Fatalf("runTestLifecycle() error = %v, want nil", err)
	}
	if got := vm.requestCall.Load(); got != 0 {
		t.Fatalf("RequestStop calls = %d, want 0 when second signal is already present", got)
	}
	if got := vm.stopCalls.Load(); got != 1 {
		t.Fatalf("Stop calls = %d, want 1", got)
	}
}

func TestVirtualizationFrameworkLifecycleStateErrorReturnsWithoutForceStop(t *testing.T) {
	states := make(chan lifecycleTestState, 1)
	states <- lifecycleTestError
	vm := &fakeVirtualizationFrameworkVM{}
	err := runTestLifecycle(context.Background(), nil, vm, states, shortLifecycleTimeouts(), virtualizationFrameworkLifecycleHooks{})
	if err == nil || !strings.Contains(err.Error(), "entered error state") {
		t.Fatalf("runTestLifecycle() error = %v, want VM error state", err)
	}
	if got := vm.stopCalls.Load(); got != 0 {
		t.Fatalf("Stop calls = %d, want no automatic force stop", got)
	}
	if got := vm.requestCall.Load(); got != 0 {
		t.Fatalf("RequestStop calls = %d, want 0", got)
	}
}

func TestVirtualizationFrameworkLifecycleClosedStateStreamWaitsWithoutSpinning(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	forceStop := make(chan struct{})
	states := make(chan lifecycleTestState)
	close(states)
	requestStarted := make(chan struct{})
	logs := make(chan string, 32)
	closedStateLogs := atomic.Int32{}
	hooks := virtualizationFrameworkLifecycleHooks{
		Logf: func(format string, args ...any) {
			message := fmt.Sprintf(format, args...)
			if strings.Contains(message, "state notifications closed") {
				closedStateLogs.Add(1)
			}
			select {
			case logs <- message:
			default:
			}
		},
	}
	vm := &fakeVirtualizationFrameworkVM{
		requestStop: func() (bool, error) {
			close(requestStarted)
			return false, nil
		},
	}
	result := make(chan error, 1)
	go func() { result <- runTestLifecycle(ctx, forceStop, vm, states, shortLifecycleTimeouts(), hooks) }()
	waitForLifecycleLog(t, logs, "state notifications closed")
	time.Sleep(25 * time.Millisecond)
	if got := closedStateLogs.Load(); got != 1 {
		t.Fatalf("closed-state log count = %d, want one notification without a busy loop", got)
	}

	cancel()
	waitLifecycleNotification(t, requestStarted, "graceful stop request")
	waitForLifecycleLog(t, logs, "guest did not accept graceful shutdown request")
	assertLifecycleStillWaiting(t, result, vm)
	close(forceStop)
	err := waitLifecycleResult(t, result)
	if err == nil || !strings.Contains(err.Error(), "state notification channel closed") {
		t.Fatalf("runTestLifecycle() error = %v, want preserved state stream closure error", err)
	}
	if got := vm.stopCalls.Load(); got != 1 {
		t.Fatalf("Stop calls = %d, want 1 after second signal", got)
	}
}

func TestVirtualizationFrameworkLifecycleForceStopErrorAndTimeout(t *testing.T) {
	stopErr := errors.New("force stop failed")
	tests := []struct {
		name    string
		stop    func() error
		want    string
		release bool
	}{
		{name: "stop error", stop: func() error { return stopErr }, want: "force stop failed"},
		{name: "stop timeout", want: "timed out", release: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			forceStop := make(chan struct{})
			startReturned := make(chan struct{})
			requestStarted := make(chan struct{})
			releaseStop := make(chan struct{})
			logs := make(chan string, 16)
			vm := &fakeVirtualizationFrameworkVM{
				start: func() error {
					close(startReturned)
					return nil
				},
				requestStop: func() (bool, error) {
					close(requestStarted)
					return false, nil
				},
			}
			if tt.stop != nil {
				vm.stop = tt.stop
			} else {
				vm.stop = func() error {
					<-releaseStop
					return nil
				}
			}
			result := make(chan error, 1)
			timeouts := virtualizationFrameworkLifecycleTimeouts{ForceStop: 30 * time.Millisecond}
			go func() {
				result <- runTestLifecycle(ctx, forceStop, vm, make(chan lifecycleTestState), timeouts, lifecycleLogHooks(logs))
			}()
			waitLifecycleNotification(t, startReturned, "VM Start completion")
			cancel()
			waitLifecycleNotification(t, requestStarted, "graceful stop request")
			waitForLifecycleLog(t, logs, "guest did not accept graceful shutdown request")
			assertLifecycleStillWaiting(t, result, vm)
			close(forceStop)
			err := waitLifecycleResult(t, result)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("runTestLifecycle() error = %v, want text %q", err, tt.want)
			}
			if got := vm.stopCalls.Load(); got != 1 {
				t.Fatalf("Stop calls = %d, want 1", got)
			}
			if tt.release {
				close(releaseStop)
			}
		})
	}
}

func TestVirtualizationFrameworkLifecycleStartFailureAndPreStartCancellation(t *testing.T) {
	startErr := errors.New("start failed")
	vm := &fakeVirtualizationFrameworkVM{start: func() error { return startErr }}
	err := runTestLifecycle(context.Background(), nil, vm, make(chan lifecycleTestState), shortLifecycleTimeouts(), virtualizationFrameworkLifecycleHooks{})
	if !errors.Is(err, startErr) {
		t.Fatalf("runTestLifecycle() error = %v, want wrapped %v", err, startErr)
	}
	if got := vm.stopCalls.Load(); got != 0 {
		t.Fatalf("Stop calls after start failure = %d, want 0", got)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	vm = &fakeVirtualizationFrameworkVM{}
	err = runTestLifecycle(ctx, nil, vm, make(chan lifecycleTestState), shortLifecycleTimeouts(), virtualizationFrameworkLifecycleHooks{})
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
	go func() {
		result <- runTestLifecycle(ctx, nil, vm, states, shortLifecycleTimeouts(), virtualizationFrameworkLifecycleHooks{})
	}()
	waitLifecycleNotification(t, startEntered, "VM Start entry")
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
	if got := vm.stopCalls.Load(); got != 0 {
		t.Fatalf("Stop calls = %d, want natural stop", got)
	}
}

func TestVirtualizationFrameworkLifecycleNaturalStopWhileRequestBlocks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	startReturned := make(chan struct{})
	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})
	requestFinished := make(chan struct{})
	states := make(chan lifecycleTestState, 1)
	vm := &fakeVirtualizationFrameworkVM{
		start: func() error {
			close(startReturned)
			return nil
		},
		requestStop: func() (bool, error) {
			close(requestStarted)
			<-releaseRequest
			close(requestFinished)
			return true, nil
		},
	}
	result := make(chan error, 1)
	go func() {
		result <- runTestLifecycle(ctx, nil, vm, states, shortLifecycleTimeouts(), virtualizationFrameworkLifecycleHooks{})
	}()
	waitLifecycleNotification(t, startReturned, "VM Start completion")
	cancel()
	waitLifecycleNotification(t, requestStarted, "graceful stop request")
	states <- lifecycleTestStopped
	if err := waitLifecycleResult(t, result); err != nil {
		t.Fatalf("runTestLifecycle() error = %v, want natural stopped state to end supervision", err)
	}
	if got := vm.stopCalls.Load(); got != 0 {
		t.Fatalf("Stop calls = %d, want natural stop", got)
	}
	close(releaseRequest)
	select {
	case <-requestFinished:
	case <-time.After(time.Second):
		t.Fatal("blocked RequestStop did not finish after release")
	}
}

func TestVirtualizationFrameworkLifecycleStoppedStateConfirmsForceStopRace(t *testing.T) {
	forceStop := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	startReturned := make(chan struct{})
	requestStarted := make(chan struct{})
	stopStarted := make(chan struct{})
	releaseStop := make(chan struct{})
	logs := make(chan string, 16)
	states := make(chan lifecycleTestState)
	vm := &fakeVirtualizationFrameworkVM{
		start: func() error {
			close(startReturned)
			return nil
		},
		requestStop: func() (bool, error) {
			close(requestStarted)
			return false, nil
		},
		stop: func() error {
			close(stopStarted)
			<-releaseStop
			return errors.New("VM already stopped")
		},
	}
	result := make(chan error, 1)
	go func() {
		result <- runTestLifecycle(ctx, forceStop, vm, states, shortLifecycleTimeouts(), lifecycleLogHooks(logs))
	}()
	waitLifecycleNotification(t, startReturned, "VM Start completion")
	cancel()
	waitLifecycleNotification(t, requestStarted, "graceful stop request")
	waitForLifecycleLog(t, logs, "guest did not accept graceful shutdown request")
	assertLifecycleStillWaiting(t, result, vm)
	close(forceStop)
	waitLifecycleNotification(t, stopStarted, "forced stop start")
	states <- lifecycleTestStopped
	err := waitLifecycleResult(t, result)
	close(releaseStop)
	if err != nil {
		t.Fatalf("runTestLifecycle() error = %v, want stopped-state confirmation to resolve force-stop race", err)
	}
}

func TestVirtualizationFrameworkDeadlineRecheckPrefersBufferedResult(t *testing.T) {
	resultCh := make(chan int, 1)
	resultCh <- 42
	states := make(chan lifecycleTestState, 1)
	states <- lifecycleTestStopped

	result, ready, stopped, stateErr := recheckVirtualizationFrameworkDeadline(
		resultCh,
		states,
		lifecycleTestStopped,
		lifecycleTestError,
	)
	if !ready || result != 42 {
		t.Fatalf("recheck result = (%d, %t), want (42, true)", result, ready)
	}
	if stopped || stateErr != nil {
		t.Fatalf("recheck state = (%t, %v), want state check skipped when result is ready", stopped, stateErr)
	}
	if len(states) != 1 {
		t.Fatalf("queued state count = %d, want stopped notification left for normal processing", len(states))
	}
}

func TestVirtualizationFrameworkForceStopStateDrainIsBounded(t *testing.T) {
	states := make(chan lifecycleTestState, virtualizationFrameworkForceStopStateCheckLimit+2)
	for i := 0; i < cap(states); i++ {
		states <- lifecycleTestRunning
	}

	stopped, stateErr := drainVirtualizationFrameworkVMStates(states, lifecycleTestStopped, lifecycleTestError)
	if stopped || stateErr != nil {
		t.Fatalf("drain states = (%t, %v), want no stop and no error", stopped, stateErr)
	}
	if got := len(states); got != 2 {
		t.Fatalf("queued state count after bounded drain = %d, want 2", got)
	}
}

func runTestLifecycle(
	ctx context.Context,
	forceStop <-chan struct{},
	vm virtualizationFrameworkVMOperations,
	states <-chan lifecycleTestState,
	timeouts virtualizationFrameworkLifecycleTimeouts,
	hooks virtualizationFrameworkLifecycleHooks,
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
		hooks,
	)
}

func shortLifecycleTimeouts() virtualizationFrameworkLifecycleTimeouts {
	return virtualizationFrameworkLifecycleTimeouts{ForceStop: 40 * time.Millisecond}
}

func lifecycleLogHooks(logs chan<- string) virtualizationFrameworkLifecycleHooks {
	return virtualizationFrameworkLifecycleHooks{
		Logf: func(format string, args ...any) {
			select {
			case logs <- fmt.Sprintf(format, args...):
			default:
			}
		},
	}
}

func waitLifecycleNotification(t *testing.T, notification <-chan struct{}, name string) {
	t.Helper()
	select {
	case <-notification:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", name)
	}
}

func waitForLifecycleLog(t *testing.T, logs <-chan string, fragment string) {
	t.Helper()
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case message := <-logs:
			if strings.Contains(message, fragment) {
				return
			}
		case <-timer.C:
			t.Fatalf("did not receive lifecycle log containing %q", fragment)
		}
	}
}

func assertLifecycleStillWaiting(t *testing.T, result <-chan error, vm *fakeVirtualizationFrameworkVM) {
	t.Helper()
	timer := time.NewTimer(40 * time.Millisecond)
	defer timer.Stop()
	select {
	case err := <-result:
		t.Fatalf("lifecycle returned before natural stop or second signal: %v", err)
	case <-timer.C:
	}
	if got := vm.stopCalls.Load(); got != 0 {
		t.Fatalf("Stop calls = %d, want no automatic force stop", got)
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
