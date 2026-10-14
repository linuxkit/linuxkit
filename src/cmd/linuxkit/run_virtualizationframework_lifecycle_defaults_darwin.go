//go:build darwin && cgo

package main

import "time"

func defaultVirtualizationFrameworkLifecycleTimeouts() virtualizationFrameworkLifecycleTimeouts {
	return virtualizationFrameworkLifecycleTimeouts{
		ForceStop: 5 * time.Second,
	}
}
