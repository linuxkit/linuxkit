package main

// Please note this file is shared between pkg/runc and pkg/containerd
// Update it in both places if you make changes

import (
	"path/filepath"
	"syscall"
)

func prepare(path string) error {
	rootfs := filepath.Join(path, "rootfs")
	if err := syscall.Mount(rootfs, rootfs, "", syscall.MS_BIND, ""); err != nil {
		return err
	}
	// remount rw
	if err := syscall.Mount("", rootfs, "", syscall.MS_REMOUNT, ""); err != nil {
		return err
	}
	return nil
}
