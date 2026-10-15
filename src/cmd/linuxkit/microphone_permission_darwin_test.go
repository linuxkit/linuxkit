//go:build darwin && cgo
// +build darwin,cgo

package main

import (
	"bytes"
	"context"
	"debug/macho"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestEnsureVirtualizationFrameworkMicrophonePermissionRejectsNilContext(t *testing.T) {
	//nolint:staticcheck // nil is the case this boundary test must reject.
	err := ensureVirtualizationFrameworkMicrophonePermission(nil)
	if err == nil || !strings.Contains(err.Error(), "requires a context") {
		t.Fatalf("permission check error = %v, want a context-required error", err)
	}
}

func TestEnsureVirtualizationFrameworkMicrophonePermissionHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := ensureVirtualizationFrameworkMicrophonePermission(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("permission check error = %v, want context.Canceled", err)
	}
}

func TestEnsureVirtualizationFrameworkMicrophonePermissionRequiresUsageDescription(t *testing.T) {
	if testBinaryHasMicrophoneUsageDescription(t) {
		t.Skip("test binary embeds NSMicrophoneUsageDescription")
	}

	err := ensureVirtualizationFrameworkMicrophonePermission(context.Background())
	if err == nil || !strings.Contains(err.Error(), "missing NSMicrophoneUsageDescription") {
		t.Fatalf("permission check error = %v, want actionable missing usage description error", err)
	}
}

func testBinaryHasMicrophoneUsageDescription(t *testing.T) bool {
	t.Helper()

	executable, err := os.Executable()
	if err != nil {
		t.Skipf("cannot inspect test binary for embedded Info.plist: %v", err)
	}
	binary, err := macho.Open(executable)
	if err != nil {
		t.Skipf("cannot inspect test binary for embedded Info.plist: %v", err)
	}
	defer func() {
		if err := binary.Close(); err != nil {
			t.Errorf("close test binary: %v", err)
		}
	}()

	plistSection := binary.Section("__info_plist")
	if plistSection == nil {
		return false
	}
	plist, err := plistSection.Data()
	if err != nil {
		t.Skipf("cannot inspect test binary Info.plist: %v", err)
	}
	key := []byte("<key>NSMicrophoneUsageDescription</key>")
	keyOffset := bytes.Index(plist, key)
	if keyOffset < 0 {
		return false
	}
	value := plist[keyOffset+len(key):]
	start := bytes.Index(value, []byte("<string>"))
	end := bytes.Index(value, []byte("</string>"))
	return start >= 0 && end > start+len("<string>")
}
