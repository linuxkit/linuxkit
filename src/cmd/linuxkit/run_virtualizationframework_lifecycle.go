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
	ForceStop time.Duration
}

func defaultVirtualizationFrameworkLifecycleTimeouts() virtualizationFrameworkLifecycleTimeouts {
	return virtualizationFrameworkLifecycleTimeouts{
		ForceStop: 5 * time.Second,
	}
}

type virtualizationFrameworkLifecycleHooks struct {
	Logf      func(string, ...any)
	OnRunning func()
}

// Bound state draining so nonterminal notifications cannot delay a forced stop.
const virtualizationFrameworkForceStopStateCheckLimit = 8

type virtualizationFrameworkStopRequestResult struct {
	accepted bool
	err      error
}

// runVirtualizationFrameworkVMLifecycle owns Start synchronously so a late
// completion cannot start a VM after the caller has stopped supervising it.
// RequestStop and Stop run asynchronously so signal handling stays responsive
// while their cgo completion handlers are pending.
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

	stateNotifications := states
	requestDone := (<-chan virtualizationFrameworkStopRequestResult)(nil)
	ctxDone := ctx.Done()
	var priorErr error
	requestStop := func() {
		ctxDone = nil
		resultCh := make(chan virtualizationFrameworkStopRequestResult, 1)
		requestDone = resultCh
		go func() {
			accepted, err := vm.RequestStop()
			resultCh <- virtualizationFrameworkStopRequestResult{accepted: accepted, err: err}
		}()
		if hooks.Logf != nil {
			hooks.Logf("shutdown requested; asking guest to stop")
		}
	}

	for {
		if channelClosed(forceStop) {
			if hooks.Logf != nil {
				hooks.Logf("second signal received; forcing virtual machine stop")
			}
			requestDone, priorErr = collectCompletedVirtualizationFrameworkStopRequest(requestDone, priorErr, hooks)
			return forceStopVirtualizationFrameworkVM(vm, stateNotifications, stoppedState, errorState, requestDone, timeouts.ForceStop, hooks, priorErr)
		}

		select {
		case state, ok := <-stateNotifications:
			if !ok {
				priorErr = errors.Join(priorErr, errors.New("virtual machine state notification channel closed before the VM stopped"))
				if hooks.Logf != nil {
					hooks.Logf("virtual machine state notifications closed; waiting for a second signal to force stop")
				}
				stateNotifications = nil
				continue
			}
			switch state {
			case stoppedState:
				requestDone, priorErr = collectCompletedVirtualizationFrameworkStopRequest(requestDone, priorErr, hooks)
				if hooks.Logf != nil {
					hooks.Logf("stopped successfully")
				}
				return priorErr
			case errorState:
				requestDone, priorErr = collectCompletedVirtualizationFrameworkStopRequest(requestDone, priorErr, hooks)
				stateErr := errors.New("virtual machine entered error state")
				if hooks.Logf != nil {
					hooks.Logf("virtual machine entered error state; ending supervision without forcing a stop")
				}
				return errors.Join(priorErr, stateErr)
			default:
				if state == runningState && hooks.OnRunning != nil {
					hooks.OnRunning()
				}
			}
		case result := <-requestDone:
			requestDone = nil
			priorErr = recordVirtualizationFrameworkStopRequest(result, priorErr, hooks)
		case <-ctxDone:
			requestStop()
		case <-forceStop:
			if hooks.Logf != nil {
				hooks.Logf("second signal received; forcing virtual machine stop")
			}
			requestDone, priorErr = collectCompletedVirtualizationFrameworkStopRequest(requestDone, priorErr, hooks)
			return forceStopVirtualizationFrameworkVM(vm, stateNotifications, stoppedState, errorState, requestDone, timeouts.ForceStop, hooks, priorErr)
		}
	}
}

func validateVirtualizationFrameworkLifecycleTimeouts(timeouts virtualizationFrameworkLifecycleTimeouts) error {
	if timeouts.ForceStop <= 0 {
		return fmt.Errorf("force stop timeout must be positive: %s", timeouts.ForceStop)
	}
	return nil
}

func recordVirtualizationFrameworkStopRequest(
	result virtualizationFrameworkStopRequestResult,
	priorErr error,
	hooks virtualizationFrameworkLifecycleHooks,
) error {
	if result.err != nil {
		if hooks.Logf != nil {
			hooks.Logf("graceful shutdown request failed; waiting for VM stop or a second signal: %v", result.err)
		}
		return errors.Join(priorErr, fmt.Errorf("could not request virtual machine stop: %w", result.err))
	}
	if !result.accepted {
		if hooks.Logf != nil {
			hooks.Logf("guest did not accept graceful shutdown request; waiting for VM stop or a second signal")
		}
		return priorErr
	}
	if hooks.Logf != nil {
		hooks.Logf("guest accepted graceful shutdown request; waiting for VM stop or a second signal")
	}
	return priorErr
}

func collectCompletedVirtualizationFrameworkStopRequest(
	requestDone <-chan virtualizationFrameworkStopRequestResult,
	priorErr error,
	hooks virtualizationFrameworkLifecycleHooks,
) (<-chan virtualizationFrameworkStopRequestResult, error) {
	if requestDone == nil {
		return nil, priorErr
	}
	select {
	case result := <-requestDone:
		return nil, recordVirtualizationFrameworkStopRequest(result, priorErr, hooks)
	default:
		return requestDone, priorErr
	}
}

func forceStopVirtualizationFrameworkVM[S virtualizationFrameworkVMState](
	vm virtualizationFrameworkVMOperations,
	states <-chan S,
	stoppedState, errorState S,
	requestDone <-chan virtualizationFrameworkStopRequestResult,
	timeout time.Duration,
	hooks virtualizationFrameworkLifecycleHooks,
	priorErr error,
) error {
	requestDone, priorErr = collectCompletedVirtualizationFrameworkStopRequest(requestDone, priorErr, hooks)
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
	requestResults := requestDone
	var stopErr error
	for {
		select {
		case err := <-stopDone:
			stopDone = nil
			if err != nil {
				stopErr = fmt.Errorf("could not force stop virtual machine: %w", err)
				continue
			}
			_, priorErr = collectCompletedVirtualizationFrameworkStopRequest(requestResults, priorErr, hooks)
			if hooks.Logf != nil {
				hooks.Logf("stopped successfully after force stop")
			}
			return priorErr
		case result := <-requestResults:
			requestResults = nil
			priorErr = recordVirtualizationFrameworkStopRequest(result, priorErr, hooks)
		case state, ok := <-stateNotifications:
			if !ok {
				priorErr = errors.Join(priorErr, errors.New("virtual machine state notification channel closed before the VM stopped"))
				stateNotifications = nil
				continue
			}
			if state == stoppedState {
				_, priorErr = collectCompletedVirtualizationFrameworkStopRequest(requestResults, priorErr, hooks)
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
					_, priorErr = collectCompletedVirtualizationFrameworkStopRequest(requestResults, priorErr, hooks)
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
				_, priorErr = collectCompletedVirtualizationFrameworkStopRequest(requestResults, priorErr, hooks)
				if hooks.Logf != nil {
					hooks.Logf("stopped successfully while force stop was completing")
				}
				return priorErr
			}
			_, priorErr = collectCompletedVirtualizationFrameworkStopRequest(requestResults, priorErr, hooks)
			if stopErr != nil {
				return errors.Join(priorErr, stopErr)
			}
			return errors.Join(priorErr, fmt.Errorf("forcing virtual machine stop timed out after %s", timeout))
		}
	}
}
func drainVirtualizationFrameworkVMStates[S virtualizationFrameworkVMState](states <-chan S, stoppedState, errorState S) (bool, error) {
	var stateErr error
	for i := 0; i < virtualizationFrameworkForceStopStateCheckLimit; i++ {
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
