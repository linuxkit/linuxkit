package main

import (
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"syscall"
)

const (
	configFile = "config"
	fsType     = "iso9660"
)

// ProviderISO is the type implementing the Provider interface for any
// disks or partions of iso9660 format that contain /config
type ProviderISO struct {
	device     string
	mountPoint string
	err        error
	data       []byte
}

func blockDevices() []string {
	return []string{
		// SCSI CD-ROM devices
		"/dev/sr[0-9]*",
		"/dev/scd[0-9]*",
		// SCSI disk devices
		"/dev/sd[a-z]*",
		// MMC block devices
		"/dev/mmcblk[0-9]*",
	}
}

// ListDisks lists all the cdroms in the system
func ListDisks() []Provider {
	providers := []Provider{}
	for _, s := range blockDevices() {
		disks, err := filepath.Glob(s)
		if err != nil {
			// Glob can only error on invalid pattern
			panic(fmt.Sprintf("Invalid glob pattern: %s", s))
		}
		for _, device := range disks {
			providers = append(providers, NewProviderISO(device))
		}
	}
	return providers
}

// NewProviderISO returns a new ProviderISO
func NewProviderISO(device string) *ProviderISO {
	mountPoint, err := ioutil.TempDir("", "mnt")
	p := ProviderISO{device, mountPoint, err, []byte{}}
	if err == nil {
		if p.err = p.mount(); p.err == nil {
			p.data, p.err = ioutil.ReadFile(path.Join(p.mountPoint, configFile))
			p.unmount()
		}
	}
	return &p
}

func (p *ProviderISO) String() string {
	return "ISO " + p.device
}

// Probe checks if the disk has the right file
func (p *ProviderISO) Probe() bool {
	return len(p.data) != 0
}

// Extract gets both the disk specific and generic userdata
func (p *ProviderISO) Extract() ([]byte, error) {
	return p.data, p.err
}

// mount mounts a disk under mountPoint
func (p *ProviderISO) mount() error {
	// We may need to poll a little for device ready
	return syscall.Mount(p.device, p.mountPoint, fsType, syscall.MS_RDONLY, "")
}

// unmount removes the mount
func (p *ProviderISO) unmount() {
	_ = syscall.Unmount(p.mountPoint, 0)
}
