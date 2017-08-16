package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"syscall"

	"github.com/vishvananda/netlink"
)

const (
	wgPath      = "/usr/bin/wg"
	nsenterPath = "/usr/bin/nsenter-net"
)

// Note these definitions are from moby/tool/src/moby/config.go and should be kept in sync

// Runtime is the type of config processed at runtime, not used to build the OCI spec
type Runtime struct {
	Mkdir      []string    `yaml:"mkdir" json:"mkdir,omitempty"`
	Interfaces []Interface `yaml:"interfaces" json:"interfaces,omitempty"`
	BindNS     *Namespaces `yaml:"bindNS" json:"bindNS,omitempty"`
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
	conf, err := ioutil.ReadFile(filepath.Join(path, "runtime.json"))
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

// prepareFilesystem sets up the mounts, before the container is created
func prepareFilesystem(path string, runtime Runtime) error {
	// execute the runtime config that should be done up front
	for _, dir := range runtime.Mkdir {
		// in future we may need to change the structure to set mode, ownership
		var mode os.FileMode = 0755
		err := os.MkdirAll(dir, mode)
		if err != nil {
			return fmt.Errorf("Cannot create directory %s: %v", dir, err)
		}
	}

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
	if err := os.Mkdir(upper, 0755); err != nil {
		return err
	}
	work := filepath.Join(tmp, "work")
	if err := os.Mkdir(work, 0755); err != nil {
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
	if err := syscall.Mount(fmt.Sprintf("/proc/%d/ns/%s", pid, ns), path, "", syscall.MS_BIND, ""); err != nil {
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
				link = &netlink.GenericLink{la, iface.Add}
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
			if err := netlink.LinkSetNsPid(link, int(pid)); err != nil {
				return fmt.Errorf("Cannot move interface %s into namespace: %v", iface.Name, err)
			}
			fmt.Fprintf(os.Stderr, "Moved interface %s to pid %d\n", iface.Name, pid)
		}
	}

	if runtime.BindNS != nil {
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
