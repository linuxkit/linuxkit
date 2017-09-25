package main

import (
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
)

// pivot_root, which runc uses normally will not work on a ramfs or tmpfs root filesystem
// You can work around this by forcing it to use move, but it is more convenient to create a
// new tmpfs root without this issue. Try to do this using as little RAM as possible by moving
// files one by one. Equivalent to a more memory efficient version of
// /bin/cp -a / /mnt 2>/dev/null
// exec /bin/busybox switch_root /mnt /sbin/init

// copies ownership and timestamps; assume you created with the right mode
func copyMetadata(info os.FileInfo, path string) error {
	// would rather use fd than path but Go makes this very difficult at present
	stat := info.Sys().(*syscall.Stat_t)
	if err := unix.Lchown(path, int(stat.Uid), int(stat.Gid)); err != nil {
		return err
	}
	timespec := []unix.Timespec{unix.Timespec(stat.Atim), unix.Timespec(stat.Mtim)}
	if err := unix.UtimesNanoAt(unix.AT_FDCWD, path, timespec, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return err
	}
	// after chown suid bits may be dropped; re-set on non symlink files
	if info.Mode()&os.ModeSymlink == 0 {
		if err := os.Chmod(path, info.Mode()); err != nil {
			return err
		}
	}
	return nil
}

func copyFS(newRoot string) error {
	// find the device of the root filesystem so we can avoid changing filesystem
	info, err := os.Stat("/")
	if err != nil {
		return err
	}
	stat := info.Sys().(*syscall.Stat_t)
	rootDev := stat.Dev

	if err = unix.Mount("rootfs", newRoot, "tmpfs", 0, ""); err != nil {
		return err
	}

	// copy directory tree first
	if err := filepath.Walk("/", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// skip non directories
		if !info.Mode().IsDir() {
			return nil
		}
		dest := filepath.Join(newRoot, path)
		// create the directory
		if path == "/" {
			// the mountpoint already exists but may have wrong mode, metadata
			if err := os.Chmod(newRoot, info.Mode()); err != nil {
				return err
			}
		} else {
			if err := os.Mkdir(dest, info.Mode()); err != nil {
				return err
			}
		}
		if err := copyMetadata(info, dest); err != nil {
			return err
		}
		// skip recurse into other filesystems
		stat := info.Sys().(*syscall.Stat_t)
		if rootDev != stat.Dev {
			return filepath.SkipDir
		}
		return nil
	}); err != nil {
		return err
	}

	buf := make([]byte, 32768)

	// now move files
	if err := filepath.Walk("/", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// skip other filesystems
		stat := info.Sys().(*syscall.Stat_t)
		if rootDev != stat.Dev && info.Mode().IsDir() {
			return filepath.SkipDir
		}
		dest := filepath.Join(newRoot, path)
		switch {
		case info.Mode().IsDir():
			// already done the directories
			return nil
		case info.Mode().IsRegular():
			// TODO support hard links (currently not handled well in initramfs)
			new, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE, info.Mode())
			if err != nil {
				return err
			}
			old, err := os.Open(path)
			if err != nil {
				return err
			}
			if _, err := io.CopyBuffer(new, old, buf); err != nil {
				return err
			}
			if err := old.Close(); err != nil {
				return err
			}
			if err := new.Close(); err != nil {
				return err
			}
			// it is ok if we do not remove all files now
			_ = os.Remove(path)
		case (info.Mode() & os.ModeSymlink) == os.ModeSymlink:
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			if err := os.Symlink(link, dest); err != nil {
				return err
			}
		case (info.Mode() & os.ModeDevice) == os.ModeDevice:
			if err := unix.Mknod(dest, uint32(info.Mode()), int(stat.Rdev)); err != nil {
				return err
			}
		case (info.Mode() & os.ModeNamedPipe) == os.ModeNamedPipe:
			// TODO support named pipes, although no real use case
			return errors.New("Unsupported named pipe on rootfs")
		case (info.Mode() & os.ModeSocket) == os.ModeSocket:
			// TODO support sockets, although no real use case
			return errors.New("Unsupported socket on rootfs")
		default:
			return errors.New("Unknown file type")
		}
		if err := copyMetadata(info, dest); err != nil {
			return err
		}
		// TODO copy extended attributes if needed
		return nil
	}); err != nil {
		return err
	}

	// chdir to the new root directory
	if err := os.Chdir(newRoot); err != nil {
		return err
	}
	// delete remaining directories in /
	if err := filepath.Walk("/", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// ignore root itself
		if path == "/" {
			return nil
		}
		switch {
		case info.Mode().IsDir():
			// skip other filesystems (ie newRoot)
			stat := info.Sys().(*syscall.Stat_t)
			if rootDev != stat.Dev {
				return filepath.SkipDir
			}
			// do our best to delete
			_ = os.RemoveAll(path)
			return filepath.SkipDir
		default:
			// should not be any left now
			_ = os.Remove(path)
			return nil
		}
	}); err != nil {
		return err
	}

	// mount --move cwd (/mnt) to /
	if err := unix.Mount(".", "/", "", unix.MS_MOVE, ""); err != nil {
		return err
	}

	// chroot to .
	if err := unix.Chroot("."); err != nil {
		return err
	}

	// chdir to "/" to fix up . and ..
	return os.Chdir("/")
}

func main() {
	// test if we want to do this, ie if tmpfs or ramfs
	// we could be booting off ISO, disk where we do not need this
	var sfs unix.Statfs_t
	if err := unix.Statfs("/", &sfs); err != nil {
		log.Fatalf("Cannot statfs /: %v", err)
	}
	const ramfsMagic = 0x858458f6
	const tmpfsMagic = 0x01021994
	if sfs.Type == ramfsMagic || sfs.Type == tmpfsMagic {
		const newRoot = "/mnt"

		if err := copyFS(newRoot); err != nil {
			log.Fatalf("Copy root failed: %v", err)
		}
	}

	// exec /sbin/init
	if err := syscall.Exec("/sbin/init", []string{"/sbin/init"}, os.Environ()); err != nil {
		log.Fatalf("Cannot exec /sbin/init")
	}
}
