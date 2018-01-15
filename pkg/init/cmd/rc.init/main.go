package main

import (
	"bufio"
	"encoding/csv"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

const (
	nodev    = unix.MS_NODEV
	noexec   = unix.MS_NOEXEC
	nosuid   = unix.MS_NOSUID
	readonly = unix.MS_RDONLY
	rec      = unix.MS_REC
	relatime = unix.MS_RELATIME
	remount  = unix.MS_REMOUNT
	shared   = unix.MS_SHARED
)

var (
	rliminf  = unix.RLIM_INFINITY
	infinity = uint64(rliminf)
)

// set as a subreaper
func subreaper() {
	err := unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, uintptr(1), 0, 0, 0)
	if err != nil {
		log.Printf("error setting as a subreaper: %v", err)
	}
}

// nothing really to error to, so just warn
func mount(source string, target string, fstype string, flags uintptr, data string) {
	err := unix.Mount(source, target, fstype, flags, data)
	if err != nil {
		log.Printf("error mounting %s to %s: %v", source, target, err)
	}
}

// in some cases, do not even log an error
func mountSilent(source string, target string, fstype string, flags uintptr, data string) {
	_ = unix.Mount(source, target, fstype, flags, data)
}

// make a character device
func mkchar(path string, mode, major, minor uint32) {
	// unix.Mknod only supports int dev numbers; this is ok for us
	dev := int(unix.Mkdev(major, minor))
	err := unix.Mknod(path, mode, dev)
	if err != nil {
		if err.Error() == "file exists" {
			return
		}
		log.Printf("error making device %s: %v", path, err)
	}
}

// symlink with error warning
func symlink(oldpath string, newpath string) {
	err := unix.Symlink(oldpath, newpath)
	if err != nil {
		log.Printf("error making symlink %s: %v", newpath, err)
	}
}

// mkdirall with warning
func mkdir(path string, perm os.FileMode) {
	err := os.MkdirAll(path, perm)
	if err != nil {
		log.Printf("error making directory %s: %v", path, err)
	}
}

// list of all enabled cgroups
func cgroupList() []string {
	list := []string{}
	f, err := os.Open("/proc/cgroups")
	if err != nil {
		log.Printf("cannot open /proc/cgroups: %v", err)
		return list
	}
	defer f.Close()
	reader := csv.NewReader(f)
	// tab delimited
	reader.Comma = '\t'
	// four fields
	reader.FieldsPerRecord = 4
	cgroups, err := reader.ReadAll()
	if err != nil {
		log.Printf("cannot parse /proc/cgroups: %v", err)
		return list
	}
	for _, cg := range cgroups {
		// see if enabled
		if cg[3] == "1" {
			list = append(list, cg[0])
		}
	}
	return list
}

// write a file, eg sysfs
func write(path string, value string) {
	err := ioutil.WriteFile(path, []byte(value), 0600)
	if err != nil {
		log.Printf("cannot write to %s: %v", path, err)
	}
}

// read a file, eg sysfs, strip whitespace, empty string if does not exist
func read(path string) string {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// read a directory
func readdir(path string) []string {
	names := []string{}
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Printf("cannot read directory %s: %v", path, err)
		return names
	}
	for _, f := range files {
		names = append(names, f.Name())
	}
	return names
}

// glob logging errors
func glob(pattern string) []string {
	files, err := filepath.Glob(pattern)
	if err != nil {
		log.Printf("error in glob %s: %v", pattern, err)
		return []string{}
	}
	return files
}

// test if a file exists
func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// modalias runs modprobe on the modalias file contents
func modalias(path string) {
	alias := read(path)
	cmd := exec.Command("/sbin/modprobe", "-abq", alias)
	// many of these error so do not report
	_ = cmd.Run()
}

func doMounts() {
	// mount proc filesystem
	mount("proc", "/proc", "proc", nodev|nosuid|noexec|relatime, "")

	// remount rootfs read only if it is not already
	mountSilent("", "/", "", remount|readonly, "")

	// mount tmpfs for /tmp and /run
	mount("tmpfs", "/run", "tmpfs", nodev|nosuid|noexec|relatime, "size=10%,mode=755")
	mount("tmpfs", "/tmp", "tmpfs", nodev|nosuid|noexec|relatime, "size=10%,mode=1777")

	// mount tmpfs for /var. This may be overmounted with a persistent filesystem later
	mount("tmpfs", "/var", "tmpfs", nodev|nosuid|noexec|relatime, "size=50%,mode=755")
	// add standard directories in /var
	mkdir("/var/cache", 0755)
	mkdir("/var/empty", 0555)
	mkdir("/var/lib", 0755)
	mkdir("/var/local", 0755)
	mkdir("/var/lock", 0755)
	mkdir("/var/log", 0755)
	mkdir("/var/opt", 0755)
	mkdir("/var/spool", 0755)
	mkdir("/var/tmp", 01777)
	symlink("/run", "/var/run")

	// mount devfs
	mount("dev", "/dev", "devtmpfs", nosuid|noexec|relatime, "size=10m,nr_inodes=248418,mode=755")
	// make minimum necessary devices
	mkchar("/dev/console", 0600, 5, 1)
	mkchar("/dev/tty1", 0620, 4, 1)
	mkchar("/dev/tty", 0666, 5, 0)
	mkchar("/dev/null", 0666, 1, 3)
	mkchar("/dev/kmsg", 0660, 1, 11)
	// make standard symlinks
	symlink("/proc/self/fd", "/dev/fd")
	symlink("/proc/self/fd/0", "/dev/stdin")
	symlink("/proc/self/fd/1", "/dev/stdout")
	symlink("/proc/self/fd/2", "/dev/stderr")
	symlink("/proc/kcore", "/dev/kcore")
	// dev mountpoints
	mkdir("/dev/mqueue", 01777)
	mkdir("/dev/shm", 01777)
	mkdir("/dev/pts", 0755)
	// mounts on /dev
	mount("mqueue", "/dev/mqueue", "mqueue", noexec|nosuid|nodev, "")
	mount("shm", "/dev/shm", "tmpfs", noexec|nosuid|nodev, "mode=1777")
	mount("devpts", "/dev/pts", "devpts", noexec|nosuid, "gid=5,mode=0620")

	// sysfs
	mount("sysfs", "/sys", "sysfs", noexec|nosuid|nodev, "")
	// some of the subsystems may not exist, so ignore errors
	mountSilent("securityfs", "/sys/kernel/security", "securityfs", noexec|nosuid|nodev, "")
	mountSilent("debugfs", "/sys/kernel/debug", "debugfs", noexec|nosuid|nodev, "")
	mountSilent("configfs", "/sys/kernel/config", "configfs", noexec|nosuid|nodev, "")
	mountSilent("fusectl", "/sys/fs/fuse/connections", "fusectl", noexec|nosuid|nodev, "")
	mountSilent("selinuxfs", "/sys/fs/selinux", "selinuxfs", noexec|nosuid, "")
	mountSilent("pstore", "/sys/fs/pstore", "pstore", noexec|nosuid|nodev, "")
	mountSilent("efivarfs", "/sys/firmware/efi/efivars", "efivarfs", noexec|nosuid|nodev, "")

	// misc /proc mounted fs
	mountSilent("binfmt_misc", "/proc/sys/fs/binfmt_misc", "binfmt_misc", noexec|nosuid|nodev, "")

	// mount cgroup root tmpfs
	mount("cgroup_root", "/sys/fs/cgroup", "tmpfs", nodev|noexec|nosuid, "mode=755,size=10m")
	// mount cgroups filesystems for all enabled cgroups
	for _, cg := range cgroupList() {
		path := filepath.Join("/sys/fs/cgroup", cg)
		mkdir(path, 0555)
		mount(cg, path, "cgroup", noexec|nosuid|nodev, cg)
	}

	// use hierarchy for memory
	write("/sys/fs/cgroup/memory/memory.use_hierarchy", "1")

	// many things assume systemd
	mkdir("/sys/fs/cgroup/systemd", 0555)
	mount("cgroup", "/sys/fs/cgroup/systemd", "cgroup", 0, "none,name=systemd")

	// make / rshared
	mount("", "/", "", rec|shared, "")
}

func doHotplug() {
	mdev := "/sbin/mdev"
	// start mdev for hotplug
	write("/proc/sys/kernel/hotplug", mdev)

	devices := "/sys/devices"
	files := readdir(devices)
	for _, f := range files {
		uevent := filepath.Join(devices, f, "uevent")
		if strings.HasPrefix(f, "usb") && exists(uevent) {
			write(uevent, "add")
		}
	}

	cmd := exec.Command(mdev, "-s")
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to run %s -s: %v", mdev, err)
	}

	// mdev only supports hot plug, so also add all existing cold plug devices
	for _, df := range glob("/sys/bus/*/devices/*/modalias") {
		modalias(df)
	}
}

func doClock() {
	cmd := exec.Command("/sbin/hwclock", "--hctosys", "--utc")
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to run hwclock: %v", err)
	}
}

func rlimit(resource int, cur, max uint64) {
	lim := unix.Rlimit{Cur: cur, Max: max}
	err := unix.Setrlimit(resource, &lim)
	if err != nil {
		log.Printf("Failed to set rlimit %d: %v", resource, err)
	}
}

func doLimits() {
	rlimit(unix.RLIMIT_NOFILE, 1048576, 1048576)
	rlimit(unix.RLIMIT_NPROC, infinity, infinity)
}

func doHostname() {
	hostname := read("/etc/hostname")
	if hostname != "" {
		if err := unix.Sethostname([]byte(hostname)); err != nil {
			log.Printf("Setting hostname failed: %v", err)
		}
	}
	hostname, err := os.Hostname()
	if err != nil {
		log.Printf("Cannot read hostname: %v", err)
		return
	}

	if hostname != "(none)" && hostname != "" {
		return
	}

	mac := read("/sys/class/net/eth0/address")
	if mac == "" {
		return
	}

	mac = strings.Replace(mac, ":", "", -1)
	if err := unix.Sethostname([]byte("linuxkit-" + mac)); err != nil {
		log.Printf("Setting hostname failed: %v", err)
	}
}

func doResolvConf() {
	// for containerizing dhcpcd and other containers that need writable /etc/resolv.conf
	// if it is a symlink (usually to /run) make the directory and empty file
	link, err := os.Readlink("/etc/resolv.conf")
	if err != nil {
		return
	}
	mkdir(filepath.Dir(link), 0755)
	write(link, "")
}

func doLoopback() {
	// TODO use netlink instead
	cmd := exec.Command("/sbin/ip", "addr", "add", "127.0.0.1/8", "dev", "lo", "brd", "+", "scope", "host")
	_ = cmd.Run()
	cmd = exec.Command("/sbin/ip", "route", "add", "127.0.0.0/8", "dev", "lo", "scope", "host")
	_ = cmd.Run()
	cmd = exec.Command("/sbin/ip", "link", "set", "lo", "up")
	_ = cmd.Run()
}

// execute scripts in /etc/init.d/ or /etc/shutdown.d. These should not block.
func runInit(path string) {
	for _, f := range readdir(path) {
		file := filepath.Join(path, f)
		fi, err := os.Stat(file)
		if err != nil {
			log.Printf("Cannot stat %s: %v", file, err)
			continue
		}
		if !fi.Mode().IsRegular() {
			continue
		}
		cmd := exec.Command(file)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
	}
}

func doReap() {
	// now reap all children
	// if we are running in a real system, init does this, but in a container it would terminate when we exit
	for {
		_, err := unix.Wait4(-1, nil, 0, nil)
		if err != nil {
			// ECHILD means no children left
			if e, ok := err.(*os.SyscallError); ok && e.Err == unix.ECHILD {
				return
			}
		}
	}
}

func unmountAll() {
	mounts, err := os.Open("/proc/mounts")
	if err != nil {
		log.Printf("Cannot open /proc/mounts: %v", err)
		return
	}
	defer mounts.Close()
	scanner := bufio.NewScanner(mounts)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, " ")
		if len(parts) > 3 {
			dest := parts[1]
			tp := parts[2]
			switch tp {
			// do not unmount tmpfs or virtual filesystems, just ones that need to write data
			case "ext2", "ext3", "ext4", "btrfs", "xfs", "vfat", "msdos", "overlay":
				if err := unix.Unmount(dest, 0); err != nil {
					log.Printf("error unmounting %s: %v", dest, err)
				}
			case "nfs", "nfs4", "cifs":
				// lazy unmount as we do not want to block on network mounts
				if err := unix.Unmount(dest, unix.MNT_DETACH); err != nil {
					log.Printf("error unmounting %s: %v", dest, err)
				}
			}
		}
	}
}

func doShutdown(action string) {
	runInit("/etc/shutdown.d")
	_ = unix.Kill(-1, unix.SIGTERM)
	time.Sleep(5 * time.Second)
	_ = unix.Kill(-1, unix.SIGKILL)
	unix.Sync()
	unmountAll()
	switch action {
	case "poweroff":
		_ = unix.Reboot(unix.LINUX_REBOOT_CMD_POWER_OFF)
	case "reboot":
		_ = unix.Reboot(unix.LINUX_REBOOT_CMD_RESTART)
	}
	// if this failed, init will try again
	os.Exit(0)
}

func main() {
	if filepath.Base(os.Args[0]) == "rc.shutdown" {
		action := "poweroff"
		if len(os.Args) > 1 {
			action = os.Args[1]
		}
		doShutdown(action)
	}
	// see if we are on a real system or in userspace
	// assume in userspace if /proc already mounted
	userspace := exists("/proc/self")

	if userspace {
		subreaper()
	} else {
		doMounts()
		doHotplug()
		doClock()
		doLoopback()
	}

	doLimits()
	doHostname()
	doResolvConf()

	runInit("/etc/init.d")

	if userspace {
		doReap()
	}
}
