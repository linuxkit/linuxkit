package main

// Please note this file is shared between pkg/runc and pkg/containerd
// Update it in both places if you make changes

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func prepare(path string) error {
	// see if we are dealing with a read only or read write container
	if _, err := os.Stat(filepath.Join(path, "lower")); err != nil {
		if os.IsNotExist(err) {
			return prepareRO(path)
		}
		return err
	}
	return prepareRW(path)
}

func prepareRO(path string) error {
	// make rootfs a mount point, as runc doesn't like it much otherwise
	rootfs := filepath.Join(path, "rootfs")
	if err := syscall.Mount(rootfs, rootfs, "", syscall.MS_BIND, ""); err != nil {
		return err
	}
	return nil
}

func prepareRW(path string) error {
	// mount a tmpfs on tmp for upper and workdirs
	// make it private as nothing else should be using this
	tmp := filepath.Join(path, "tmp")
	if err := syscall.Mount("tmpfs", tmp, "tmpfs", 0, "size=10%"); err != nil {
		return err
	}
	// make it private as nothing else should be using this
	if err := syscall.Mount("", tmp, "", syscall.MS_REMOUNT|syscall.MS_PRIVATE, ""); err != nil {
		return err
	}
	upper := filepath.Join(tmp, "upper")
	// make the mount points
	if err := os.Mkdir(upper, 0744); err != nil {
		return err
	}
	work := filepath.Join(tmp, "work")
	if err := os.Mkdir(work, 0744); err != nil {
		return err
	}
	lower := filepath.Join(path, "lower")
	rootfs := filepath.Join(path, "rootfs")
	opt := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lower, upper, work)
	if err := syscall.Mount("overlay", rootfs, "overlay", 0, opt); err != nil {
		return err
	}
	return nil
}

// cleanup functions are best efforts only, mainly for rw onboot containers
func cleanup(path string) {
	// see if we are dealing with a read only or read write container
	if _, err := os.Stat(filepath.Join(path, "lower")); err != nil {
		cleanupRO(path)
	} else {
		cleanupRW(path)
	}
}

func cleanupRO(path string) {
	// remove the bind mount
	rootfs := filepath.Join(path, "rootfs")
	_ = syscall.Unmount(rootfs, 0)
}

func cleanupRW(path string) {
	// remove the overlay mount
	rootfs := filepath.Join(path, "rootfs")
	_ = os.RemoveAll(rootfs)
	_ = syscall.Unmount(rootfs, 0)
	// remove the tmpfs
	tmp := filepath.Join(path, "tmp")
	_ = os.RemoveAll(tmp)
	_ = syscall.Unmount(tmp, 0)
}
