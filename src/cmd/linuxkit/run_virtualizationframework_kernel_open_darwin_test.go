//go:build darwin

package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

const kernelFIFOChildPathEnv = "LINUXKIT_KERNEL_FIFO_CHILD_PATH"

func TestOpenVirtualizationFrameworkKernelFileRejectsFIFOWithoutWriter(t *testing.T) {
	if fifoPath := os.Getenv(kernelFIFOChildPathEnv); fifoPath != "" {
		file, err := openVirtualizationFrameworkKernelFile(fifoPath)
		if file != nil {
			_ = file.Close()
		}
		if err == nil || !strings.Contains(err.Error(), "not a regular file") {
			t.Fatalf("openVirtualizationFrameworkKernelFile() error = %v, want non-regular-file error", err)
		}
		return
	}

	fifoPath := filepath.Join(t.TempDir(), "kernel-fifo")
	if err := syscall.Mkfifo(fifoPath, 0600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestOpenVirtualizationFrameworkKernelFileRejectsFIFOWithoutWriter$")
	cmd.Env = append(os.Environ(), kernelFIFOChildPathEnv+"="+fifoPath)
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("opening FIFO without a writer hung past the deadline; output: %s", output)
	}
	if err != nil {
		t.Fatalf("child rejected FIFO incorrectly: %v\n%s", err, output)
	}
}

func TestOpenVirtualizationFrameworkKernelFileRejectsDevice(t *testing.T) {
	path := os.DevNull
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat device %q: %v", path, err)
	}
	if info.Mode().IsRegular() {
		t.Skipf("%q is not reported as a device on this host", path)
	}

	if _, err := openVirtualizationFrameworkKernelFile(path); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("openVirtualizationFrameworkKernelFile() error = %v, want non-regular-file error", err)
	}
}

func TestOpenVirtualizationFrameworkKernelFileSetsCloseOnExec(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kernel")
	if err := os.WriteFile(path, []byte("kernel"), 0600); err != nil {
		t.Fatal(err)
	}
	file, err := openVirtualizationFrameworkKernelFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	flags, err := unix.FcntlInt(file.Fd(), unix.F_GETFD, 0)
	if err != nil {
		t.Fatalf("read kernel descriptor flags: %v", err)
	}
	if flags&unix.FD_CLOEXEC == 0 {
		t.Fatal("kernel descriptor is not close-on-exec")
	}
}
