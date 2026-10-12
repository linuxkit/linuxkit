package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenVirtualizationFrameworkKernelFileRejectsDirectory(t *testing.T) {
	path := t.TempDir()
	if _, err := openVirtualizationFrameworkKernelFile(path); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("openVirtualizationFrameworkKernelFile() error = %v, want non-regular-file error", err)
	}
}

func TestOpenVirtualizationFrameworkKernelFileAllowsSymlinkToRegularFile(t *testing.T) {
	workDir := t.TempDir()
	target := filepath.Join(workDir, "kernel")
	link := filepath.Join(workDir, "kernel-link")
	want := []byte("regular kernel contents")
	if err := os.WriteFile(target, want, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("cannot create symlink on this platform: %v", err)
	}

	file, err := openVirtualizationFrameworkKernelFile(link)
	if err != nil {
		t.Fatalf("openVirtualizationFrameworkKernelFile() rejected symlink to regular file: %v", err)
	}
	t.Cleanup(func() {
		if err := file.Close(); err != nil {
			t.Errorf("close kernel file: %v", err)
		}
	})

	info, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() {
		t.Fatalf("opened mode = %s, want regular file", info.Mode())
	}
	got, err := os.ReadFile(link)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("symlink contents = %q, want %q", got, want)
	}
}

func TestPrepareVirtualizationFrameworkKernelDecodesGzipThroughSymlink(t *testing.T) {
	workDir := t.TempDir()
	target := filepath.Join(workDir, "kernel.gz")
	link := filepath.Join(workDir, "kernel-link")
	tmpDir := filepath.Join(workDir, "tmp")
	if err := os.Mkdir(tmpDir, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", tmpDir)

	want := testARM64Image()
	if err := os.WriteFile(target, gzipBytes(t, want), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("cannot create symlink on this platform: %v", err)
	}

	prepared, cleanup, err := prepareVirtualizationFrameworkKernel(link)
	if err != nil {
		t.Fatalf("prepareVirtualizationFrameworkKernel() rejected gzip symlink: %v", err)
	}
	defer cleanup()
	if prepared == link {
		t.Fatal("gzip symlink was not decoded")
	}
	got, err := os.ReadFile(prepared)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("decoded kernel does not match gzip target")
	}
}
