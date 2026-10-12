package main

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type virtualizationFrameworkVMOperations interface {
	Start() error
	RequestStop() (bool, error)
	Stop() error
}

type virtualizationFrameworkVMState interface {
	~int
}

type virtualizationFrameworkLifecycleTimeouts struct {
	GracefulShutdown time.Duration
	ForceStop        time.Duration
}

func defaultVirtualizationFrameworkLifecycleTimeouts() virtualizationFrameworkLifecycleTimeouts {
	return virtualizationFrameworkLifecycleTimeouts{
		GracefulShutdown: 5 * time.Second,
		ForceStop:        5 * time.Second,
	}
}

type virtualizationFrameworkLifecycleHooks struct {
	Logf      func(string, ...any)
	OnRunning func()
}

// Bound state draining so a producer that keeps sending nonterminal states
// cannot hold shutdown past its deadline.
const virtualizationFrameworkDeadlineStateCheckLimit = 8

// runVirtualizationFrameworkVMLifecycle owns Start synchronously so a late
// completion cannot start a VM after the caller has stopped supervising it.
// RequestStop and Stop run asynchronously because their cgo completion waits
// must not make signal handling unbounded.
func runVirtualizationFrameworkVMLifecycle[S virtualizationFrameworkVMState](
	ctx context.Context,
	forceStop <-chan struct{},
	vm virtualizationFrameworkVMOperations,
	states <-chan S,
	stoppedState, runningState, errorState S,
	timeouts virtualizationFrameworkLifecycleTimeouts,
	hooks virtualizationFrameworkLifecycleHooks,
) error {
	if err := validateVirtualizationFrameworkLifecycleTimeouts(timeouts); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := vm.Start(); err != nil {
		return fmt.Errorf("could not start virtual machine: %w", err)
	}
	if hooks.Logf != nil {
		hooks.Logf("virtual machine start completed")
	}

	for {
		if ctx.Err() != nil || channelClosed(forceStop) {
			return stopVirtualizationFrameworkVM(forceStop, vm, states, stoppedState, runningState, errorState, timeouts, hooks, nil)
		}

		select {
		case state, ok := <-states:
			if !ok {
				stateErr := errors.New("virtual machine state notification channel closed before the VM stopped")
				return stopVirtualizationFrameworkVM(forceStop, vm, states, stoppedState, runningState, errorState, timeouts, hooks, stateErr)
			}
			switch state {
			case stoppedState:
				if hooks.Logf != nil {
					hooks.Logf("stopped successfully")
				}
				return nil
			case errorState:
				stateErr := errors.New("virtual machine entered error state")
				return stopVirtualizationFrameworkVM(forceStop, vm, states, stoppedState, runningState, errorState, timeouts, hooks, stateErr)
			default:
				if state == runningState && hooks.OnRunning != nil {
					hooks.OnRunning()
				}
			}
		case <-ctx.Done():
			return stopVirtualizationFrameworkVM(forceStop, vm, states, stoppedState, runningState, errorState, timeouts, hooks, nil)
		case <-forceStop:
			return stopVirtualizationFrameworkVM(forceStop, vm, states, stoppedState, runningState, errorState, timeouts, hooks, nil)
		}
	}
}

func validateVirtualizationFrameworkLifecycleTimeouts(timeouts virtualizationFrameworkLifecycleTimeouts) error {
	if timeouts.GracefulShutdown <= 0 {
		return fmt.Errorf("graceful shutdown timeout must be positive: %s", timeouts.GracefulShutdown)
	}
	if timeouts.ForceStop <= 0 {
		return fmt.Errorf("force stop timeout must be positive: %s", timeouts.ForceStop)
	}
	return nil
}

func stopVirtualizationFrameworkVM[S virtualizationFrameworkVMState](
	forceStop <-chan struct{},
	vm virtualizationFrameworkVMOperations,
	states <-chan S,
	stoppedState, runningState, errorState S,
	timeouts virtualizationFrameworkLifecycleTimeouts,
	hooks virtualizationFrameworkLifecycleHooks,
	priorErr error,
) error {
	if hooks.Logf != nil {
		hooks.Logf("shutdown requested")
	}

	if channelClosed(forceStop) {
		if hooks.Logf != nil {
			hooks.Logf("second signal received; forcing virtual machine stop")
		}
		return forceStopVirtualizationFrameworkVM(vm, states, stoppedState, errorState, timeouts.ForceStop, hooks, priorErr)
	}

	graceDeadline := time.Now().Add(timeouts.GracefulShutdown)
	type requestResult struct {
		accepted bool
		err      error
	}
	requestDone := make(chan requestResult, 1)
	go func() {
		accepted, err := vm.RequestStop()
		requestDone <- requestResult{accepted: accepted, err: err}
	}()

	requestTimer := time.NewTimer(time.Until(graceDeadline))
	defer requestTimer.Stop()
	handleRequestResult := func(result requestResult) error {
		if result.err != nil {
			priorErr = errors.Join(priorErr, fmt.Errorf("could not request virtual machine stop: %w", result.err))
			return forceStopVirtualizationFrameworkVM(vm, states, stoppedState, errorState, timeouts.ForceStop, hooks, priorErr)
		}
		if !result.accepted {
			if hooks.Logf != nil {
				hooks.Logf("guest did not accept shutdown request; forcing virtual machine stop")
			}
			return forceStopVirtualizationFrameworkVM(vm, states, stoppedState, errorState, timeouts.ForceStop, hooks, priorErr)
		}
		if hooks.Logf != nil {
			hooks.Logf("guest accepted shutdown request; waiting up to %s", time.Until(graceDeadline).Round(time.Millisecond))
		}
		stopped, forced, stateErr := waitVirtualizationFrameworkVMStop(states, forceStop, graceDeadline, stoppedState, runningState, errorState, hooks)
		if stopped {
			return priorErr
		}
		priorErr = errors.Join(priorErr, stateErr)
		if forced || channelClosed(forceStop) {
			if hooks.Logf != nil {
				hooks.Logf("second signal received; forcing virtual machine stop")
			}
		} else if stateErr == nil && hooks.Logf != nil {
			hooks.Logf("graceful shutdown timed out; forcing virtual machine stop")
		}
		return forceStopVirtualizationFrameworkVM(vm, states, stoppedState, errorState, timeouts.ForceStop, hooks, priorErr)
	}
	for {
		select {
		case result := <-requestDone:
			return handleRequestResult(result)
		case state, ok := <-states:
			if !ok {
				priorErr = errors.Join(priorErr, errors.New("virtual machine state notification channel closed before the VM stopped"))
				return forceStopVirtualizationFrameworkVM(vm, states, stoppedState, errorState, timeouts.ForceStop, hooks, priorErr)
			}
			switch state {
			case stoppedState:
				if hooks.Logf != nil {
					hooks.Logf("virtual machine stopped")
				}
				return priorErr
			case errorState:
				priorErr = errors.Join(priorErr, errors.New("virtual machine entered error state during shutdown"))
				return forceStopVirtualizationFrameworkVM(vm, states, stoppedState, errorState, timeouts.ForceStop, hooks, priorErr)
			default:
				if state == runningState && hooks.OnRunning != nil {
					hooks.OnRunning()
				}
			}
		case <-forceStop:
			if hooks.Logf != nil {
				hooks.Logf("second signal received; forcing virtual machine stop")
			}
			return forceStopVirtualizationFrameworkVM(vm, states, stoppedState, errorState, timeouts.ForceStop, hooks, priorErr)
		case <-requestTimer.C:
			if result, ready, stopped, stateErr := recheckVirtualizationFrameworkDeadline(requestDone, states, stoppedState, errorState); ready {
				return handleRequestResult(result)
			} else if stopped {
				return errors.Join(priorErr, stateErr)
			} else {
				priorErr = errors.Join(priorErr, stateErr)
			}
			priorErr = errors.Join(priorErr, fmt.Errorf("virtual machine stop request timed out after %s", timeouts.GracefulShutdown))
			if hooks.Logf != nil {
				hooks.Logf("virtual machine stop request timed out; forcing virtual machine stop")
			}
			return forceStopVirtualizationFrameworkVM(vm, states, stoppedState, errorState, timeouts.ForceStop, hooks, priorErr)
		}
	}
}

func forceStopVirtualizationFrameworkVM[S virtualizationFrameworkVMState](
	vm virtualizationFrameworkVMOperations,
	states <-chan S,
	stoppedState, errorState S,
	timeout time.Duration,
	hooks virtualizationFrameworkLifecycleHooks,
	priorErr error,
) error {
	if stopped, stateErr := drainVirtualizationFrameworkVMStates(states, stoppedState, errorState); stopped {
		return errors.Join(priorErr, stateErr)
	} else if stateErr != nil {
		priorErr = errors.Join(priorErr, stateErr)
	}

	stopDeadline := time.Now().Add(timeout)
	stopDone := make(chan error, 1)
	go func() { stopDone <- vm.Stop() }()
	stopTimer := time.NewTimer(time.Until(stopDeadline))
	defer stopTimer.Stop()
	stateNotifications := states
	var stopErr error
	for {
		select {
		case err := <-stopDone:
			stopDone = nil
			if err != nil {
				stopErr = fmt.Errorf("could not force stop virtual machine: %w", err)
				continue
			}
			if hooks.Logf != nil {
				hooks.Logf("stopped successfully after force stop")
			}
			return priorErr
		case state, ok := <-stateNotifications:
			if !ok {
				priorErr = errors.Join(priorErr, errors.New("virtual machine state notification channel closed before the VM stopped"))
				stateNotifications = nil
				continue
			}
			if state == stoppedState {
				if hooks.Logf != nil {
					hooks.Logf("stopped successfully while force stop was completing")
				}
				return priorErr
			}
			if state == errorState {
				priorErr = errors.Join(priorErr, errors.New("virtual machine entered error state during force stop"))
			}
		case <-stopTimer.C:
			result, ready, stopped, stateErr := recheckVirtualizationFrameworkDeadline(stopDone, states, stoppedState, errorState)
			if ready {
				stopDone = nil
				if result == nil {
					if hooks.Logf != nil {
						hooks.Logf("stopped successfully after force stop")
					}
					return priorErr
				}
				stopErr = fmt.Errorf("could not force stop virtual machine: %w", result)
				// A stopped notification resolves Stop errors caused by a natural
				// stop racing the forced-stop call.
				stoppedAfterError, stopStateErr := drainVirtualizationFrameworkVMStates(states, stoppedState, errorState)
				stopped = stopped || stoppedAfterError
				stateErr = errors.Join(stateErr, stopStateErr)
			}
			priorErr = errors.Join(priorErr, stateErr)
			if stopped {
				if hooks.Logf != nil {
					hooks.Logf("stopped successfully while force stop was completing")
				}
				return priorErr
			}
			if stopErr != nil {
				return errors.Join(priorErr, stopErr)
			}
			return errors.Join(priorErr, fmt.Errorf("forcing virtual machine stop timed out after %s", timeout))
		}
	}
}

func waitVirtualizationFrameworkVMStop[S virtualizationFrameworkVMState](
	states <-chan S,
	forceStop <-chan struct{},
	deadline time.Time,
	stoppedState, runningState, errorState S,
	hooks virtualizationFrameworkLifecycleHooks,
) (stopped, forced bool, err error) {
	timer := time.NewTimer(time.Until(deadline))
	defer timer.Stop()
	for {
		select {
		case state, ok := <-states:
			if !ok {
				return false, false, errors.New("virtual machine state notification channel closed before the VM stopped")
			}
			switch state {
			case stoppedState:
				if hooks.Logf != nil {
					hooks.Logf("stopped successfully")
				}
				return true, false, nil
			case errorState:
				return false, false, errors.New("virtual machine entered error state during shutdown")
			default:
				if state == runningState && hooks.OnRunning != nil {
					hooks.OnRunning()
				}
			}
		case <-forceStop:
			return false, true, nil
		case <-timer.C:
			return false, false, nil
		}
	}
}

func drainVirtualizationFrameworkVMStates[S virtualizationFrameworkVMState](states <-chan S, stoppedState, errorState S) (bool, error) {
	var stateErr error
	for i := 0; i < virtualizationFrameworkDeadlineStateCheckLimit; i++ {
		select {
		case state, ok := <-states:
			if !ok {
				return false, errors.Join(stateErr, errors.New("virtual machine state notification channel closed before the VM stopped"))
			}
			if state == stoppedState {
				return true, stateErr
			}
			if state == errorState && stateErr == nil {
				stateErr = errors.New("virtual machine entered error state during shutdown")
			}
		default:
			return false, stateErr
		}
	}
	return false, stateErr
}

func recheckVirtualizationFrameworkDeadline[T any, S virtualizationFrameworkVMState](
	resultCh <-chan T,
	states <-chan S,
	stoppedState, errorState S,
) (result T, resultReady, stopped bool, stateErr error) {
	select {
	case result, ok := <-resultCh:
		if ok {
			return result, true, false, nil
		}
	default:
	}
	stopped, stateErr = drainVirtualizationFrameworkVMStates(states, stoppedState, errorState)
	return result, false, stopped, stateErr
}

func channelClosed(ch <-chan struct{}) bool {
	if ch == nil {
		return false
	}
	select {
	case <-ch:
		return true
	default:
		return false
	}
}
