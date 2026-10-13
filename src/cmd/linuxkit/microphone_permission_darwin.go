//go:build darwin && cgo
// +build darwin,cgo

package main

/*
#cgo darwin CFLAGS: -mmacosx-version-min=12 -x objective-c -fno-objc-arc
#cgo darwin LDFLAGS: -mmacosx-version-min=12 -framework AVFoundation -framework Foundation
#include "microphone_permission_darwin.h"
*/
import "C"

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const microphonePermissionPollInterval = 50 * time.Millisecond

// ensureVirtualizationFrameworkMicrophonePermission requests host microphone
// permission before a VM configured with host audio input is started.
func ensureVirtualizationFrameworkMicrophonePermission(ctx context.Context) error {
	if ctx == nil {
		return errors.New("microphone permission check requires a context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if C.LKMicrophoneUsageDescriptionAvailable() == 0 {
		return errors.New("linuxkit is missing NSMicrophoneUsageDescription in its embedded Info.plist; rebuild with the macOS CLI build target")
	}

	switch status := int(C.LKMicrophoneAuthorizationStatus()); status {
	case C.LKMicrophoneAuthorizationAuthorized:
		return nil
	case C.LKMicrophoneAuthorizationDenied:
		return microphonePermissionDeniedError()
	case C.LKMicrophoneAuthorizationRestricted:
		return errors.New("host microphone access is restricted by macOS policy; remove the restriction or ask the device administrator")
	case C.LKMicrophoneAuthorizationNotDetermined:
		return requestVirtualizationFrameworkMicrophonePermission(ctx)
	default:
		return fmt.Errorf("macOS returned an unknown host microphone authorization status: %d", status)
	}
}

func requestVirtualizationFrameworkMicrophonePermission(ctx context.Context) error {
	request := C.LKBeginMicrophoneAuthorizationRequest()
	if request == nil {
		return errors.New("macOS could not start the host microphone permission request")
	}
	defer C.LKReleaseMicrophoneAuthorizationRequest(request)

	timer := time.NewTicker(microphonePermissionPollInterval)
	defer timer.Stop()

	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("waiting for host microphone permission: %w", err)
		}

		switch result := int(C.LKMicrophoneAuthorizationRequestResult(request)); result {
		case C.LKMicrophoneRequestGranted:
			return nil
		case C.LKMicrophoneRequestDenied:
			return microphonePermissionDeniedError()
		case C.LKMicrophoneRequestPending:
			select {
			case <-ctx.Done():
				return fmt.Errorf("waiting for host microphone permission: %w", ctx.Err())
			case <-timer.C:
			}
		default:
			return fmt.Errorf("macOS returned an invalid host microphone request result: %d", result)
		}
	}
}

func microphonePermissionDeniedError() error {
	return errors.New("host microphone access was denied; enable linuxkit or the app that launched it under System Settings > Privacy & Security > Microphone, then run the command again")
}
