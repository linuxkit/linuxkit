package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/opencontainers/runtime-spec/specs-go"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

// Note these definitions are from src/moby/config.go and should be kept in sync

// Runtime is the type of config processed at runtime, not used to build the OCI spec
type Runtime struct {
	Cgroups    []string      `yaml:"cgroups" json:"cgroups,omitempty"`
	Mounts     []specs.Mount `yaml:"mounts" json:"mounts,omitempty"`
	Mkdir      []string      `yaml:"mkdir" json:"mkdir,omitempty"`
	Interfaces []Interface   `yaml:"interfaces" json:"interfaces,omitempty"`
	BindNS     Namespaces    `yaml:"bindNS" json:"bindNS,omitempty"`
	Namespace  string        `yaml:"namespace,omitempty" json:"namespace,omitempty"`
}

// Namespaces is the type for configuring paths to bind namespaces
type Namespaces struct {
	Cgroup string `yaml:"cgroup" json:"cgroup,omitempty"`
	Ipc    string `yaml:"ipc" json:"ipc,omitempty"`
	Mnt    string `yaml:"mnt" json:"mnt,omitempty"`
	Net    string `yaml:"net" json:"net,omitempty"`
	Pid    string `yaml:"pid" json:"pid,omitempty"`
	User   string `yaml:"user" json:"user,omitempty"`
	Uts    string `yaml:"uts" json:"uts,omitempty"`
}

// Interface is the runtime config for network interfaces
type Interface struct {
	Name         string `yaml:"name" json:"name,omitempty"`
	Add          string `yaml:"add" json:"add,omitempty"`
	Peer         string `yaml:"peer" json:"peer,omitempty"`
	CreateInRoot bool   `yaml:"createInRoot" json:"createInRoot"`
}

func getRuntimeConfig(path string) Runtime {
	var runtime Runtime
	conf, err := os.ReadFile(filepath.Join(path, "runtime.json"))
	if err != nil {
		// if it does not exist it is fine to return an empty runtime, to not do anything
		if os.IsNotExist(err) {
			return runtime
		}
		log.Fatalf("Cannot read runtime config: %v", err)
	}
	if err := json.Unmarshal(conf, &runtime); err != nil {
		log.Fatalf("Cannot parse runtime config: %v", err)
	}
	return runtime
}

// parseMountOptions takes fstab style mount options and parses them for
// use with a standard mount() syscall
func parseMountOptions(options []string) (int, string) {
	var (
		flag int
		data []string
	)
	flags := map[string]struct {
		clear bool
		flag  int
	}{
		"async":         {true, unix.MS_SYNCHRONOUS},
		"atime":         {true, unix.MS_NOATIME},
		"bind":          {false, unix.MS_BIND},
		"defaults":      {false, 0},
		"dev":           {true, unix.MS_NODEV},
		"diratime":      {true, unix.MS_NODIRATIME},
		"dirsync":       {false, unix.MS_DIRSYNC},
		"exec":          {true, unix.MS_NOEXEC},
		"mand":          {false, unix.MS_MANDLOCK},
		"noatime":       {false, unix.MS_NOATIME},
		"nodev":         {false, unix.MS_NODEV},
		"nodiratime":    {false, unix.MS_NODIRATIME},
		"noexec":        {false, unix.MS_NOEXEC},
		"nomand":        {true, unix.MS_MANDLOCK},
		"norelatime":    {true, unix.MS_RELATIME},
		"nostrictatime": {true, unix.MS_STRICTATIME},
		"nosuid":        {false, unix.MS_NOSUID},
		"private":       {false, unix.MS_PRIVATE},
		"rbind":         {false, unix.MS_BIND | unix.MS_REC},
		"relatime":      {false, unix.MS_RELATIME},
		"remount":       {false, unix.MS_REMOUNT},
		"ro":            {false, unix.MS_RDONLY},
		"rw":            {true, unix.MS_RDONLY},
		"shared":        {false, unix.MS_SHARED},
		"slave":         {false, unix.MS_SLAVE},
		"strictatime":   {false, unix.MS_STRICTATIME},
		"suid":          {true, unix.MS_NOSUID},
		"sync":          {false, unix.MS_SYNCHRONOUS},
		"unbindable":    {false, unix.MS_UNBINDABLE},
	}
	for _, o := range options {
		// If the option does not exist in the flags table or the flag
		// is not supported on the platform,
		// then it is a data value for a specific fs type
		if f, exists := flags[o]; exists && f.flag != 0 {
			if f.clear {
				flag &^= f.flag
			} else {
				flag |= f.flag
			}
		} else {
			data = append(data, o)
		}
	}
	return flag, strings.Join(data, ",")
}

// newCgroup creates a cgroup (ie directory)
// we could use github.com/containerd/cgroups but it has a lot of deps and this is just a sugary mkdir
func newCgroup(cgroup string) error {
	v2, err := isCgroupV2()
	if err != nil {
		return err
	}
	if v2 {
		// a cgroupv2 cgroup is a single directory
		if err := os.MkdirAll(filepath.Join("/sys/fs/cgroup", cgroup), 0755); err != nil {
			log.Printf("cgroup error: %v", err)
		}
		return nil
	}
	// a cgroupv1 cgroup is a directory under all directories in /sys/fs/cgroup
	dirs, err := os.ReadDir("/sys/fs/cgroup")
	if err != nil {
		return err
	}

	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}
		if err := os.MkdirAll(filepath.Join("/sys/fs/cgroup", dir.Name(), cgroup), 0755); err != nil {
			log.Printf("cgroup error: %v", err)
		}
	}

	return nil
}

func isCgroupV2() (bool, error) {
	_, err := os.Stat("/sys/fs/cgroup/cgroup.controllers")
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// prepareFilesystem sets up the mounts and cgroups, before the container is created
func prepareFilesystem(path string, runtime Runtime) error {
	// execute the runtime config that should be done up front
	// we execute Mounts before Mkdir so you can make a directory under a mount
	// but we do mkdir of the destination path in case missing
	rootfs := filepath.Join(path, "rootfs")
	makeAbsolute := func(dir string) string {
		if filepath.IsAbs(dir) {
			return dir
		}
		// relative paths are relative to rootfs of container
		return filepath.Join(rootfs, dir)
	}

	for _, mount := range runtime.Mounts {
		const mode os.FileMode = 0755
		dir := makeAbsolute(mount.Destination)
		err := os.MkdirAll(dir, mode)
		if err != nil {
			return fmt.Errorf("Cannot create directory for mount destination %s: %v", dir, err)
		}
		// also mkdir upper and work directories on overlay
		for _, o := range mount.Options {
			eq := strings.SplitN(o, "=", 2)
			if len(eq) == 2 && (eq[0] == "upperdir" || eq[0] == "workdir") {
				err := os.MkdirAll(eq[1], mode)
				if err != nil {
					return fmt.Errorf("Cannot create directory for overlay %s=%s: %v", eq[0], eq[1], err)
				}
			}
		}
		opts, data := parseMountOptions(mount.Options)
		if err := unix.Mount(mount.Source, dir, mount.Type, uintptr(opts), data); err != nil {
			return fmt.Errorf("Failed to mount %s: %v", mount.Source, err)
		}
	}
	for _, dir := range runtime.Mkdir {
		// in future we may need to change the structure to set mode, ownership
		const mode os.FileMode = 0755
		dir = makeAbsolute(dir)
		err := os.MkdirAll(dir, mode)
		if err != nil {
			return fmt.Errorf("Cannot create directory %s: %v", dir, err)
		}
	}

	for _, cgroup := range runtime.Cgroups {
		// currently no way to specify resource limits on new cgroups at creation time
		if err := newCgroup(cgroup); err != nil {
			return fmt.Errorf("Cannot create cgroup %s: %v", cgroup, err)
		}
	}

	return nil
}

// bind mount a namespace file
func bindNS(ns string, path string, pid int) error {
	if path == "" {
		return nil
	}
	// the path and file need to exist for the bind to succeed, so try to create
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("Cannot create leading directories %s for bind mount destination: %v", dir, err)
	}
	fi, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("Cannot create a mount point for namespace bind at %s: %v", path, err)
	}
	if err := fi.Close(); err != nil {
		return err
	}
	if err := unix.Mount(fmt.Sprintf("/proc/%d/ns/%s", pid, ns), path, "", unix.MS_BIND, ""); err != nil {
		return fmt.Errorf("Failed to bind %s namespace at %s: %v", ns, path, err)
	}
	return nil
}

// prepareProcess sets up anything that needs to be done after the container process is created, but before it runs
// for example networking
func prepareProcess(pid int, runtime Runtime) error {
	for _, iface := range runtime.Interfaces {
		if iface.Name == "" {
			return fmt.Errorf("Interface requires a name")
		}

		var link netlink.Link
		var ns interface{} = netlink.NsPid(pid)
		var move bool
		var err error

		if iface.Peer != "" && iface.Add == "" {
			// must be a veth if specify peer
			iface.Add = "veth"
		}

		// if create in root is set, create in root namespace first, then move
		// also do the same for a veth pair
		if iface.CreateInRoot || iface.Add == "veth" {
			ns = nil
			move = true
		}

		if iface.Add != "" {
			switch iface.Add {
			case "veth":
				if iface.Peer == "" {
					return fmt.Errorf("Creating a veth pair %s requires a peer to be set", iface.Name)
				}
				la := netlink.LinkAttrs{Name: iface.Name, Namespace: ns}
				link = &netlink.Veth{LinkAttrs: la, PeerName: iface.Peer}
			default:
				// no special creation options needed
				la := netlink.LinkAttrs{Name: iface.Name, Namespace: ns}
				link = &netlink.GenericLink{LinkAttrs: la, LinkType: iface.Add}
			}
			if err := netlink.LinkAdd(link); err != nil {
				return fmt.Errorf("Link add %s of type %s failed: %v", iface.Name, iface.Add, err)
			}
			fmt.Fprintf(os.Stderr, "Created interface %s type %s\n", iface.Name, iface.Add)
		} else {
			// find existing interface
			link, err = netlink.LinkByName(iface.Name)
			if err != nil {
				return fmt.Errorf("Cannot find interface %s: %v", iface.Name, err)
			}
			// then move into namespace
			move = true
		}
		if move {
			if err := netlink.LinkSetNsPid(link, pid); err != nil {
				return fmt.Errorf("Cannot move interface %s into namespace: %v", iface.Name, err)
			}
			fmt.Fprintf(os.Stderr, "Moved interface %s to pid %d\n", iface.Name, pid)
		}
	}

	binds := []struct {
		ns   string
		path string
	}{
		{"cgroup", runtime.BindNS.Cgroup},
		{"ipc", runtime.BindNS.Ipc},
		{"mnt", runtime.BindNS.Mnt},
		{"net", runtime.BindNS.Net},
		{"pid", runtime.BindNS.Pid},
		{"user", runtime.BindNS.User},
		{"uts", runtime.BindNS.Uts},
	}

	for _, b := range binds {
		if err := bindNS(b.ns, b.path, pid); err != nil {
			return err
		}
	}

	return nil
}

// cleanup functions are best efforts only, mainly for rw onboot containers
func cleanup(path string) {
	// remove the root mount
	rootfs := filepath.Join(path, "rootfs")
	_ = unix.Unmount(rootfs, 0)
}
