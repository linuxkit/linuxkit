package main

import (
	"fmt"
	"io/ioutil"
	"os"
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
	configInfo os.FileInfo
	configPath string
	data       []byte
	err        error
}

func blockDeviceTypes() []string {
	return []string{
		// SCSI CD-ROM devices
		"/dev/sr[0-9]*",
		"/dev/scd[0-9]*",
		// MMC block devices
		"/dev/mmcblk[0-9]*",
		// SCSI disk devices
		"/dev/sd[a-z]*",
	}
}

// ListBlockDevices retuns all block devices as list of providers
func ListBlockDevices() []Provider {
	allDevices := []Provider{}
	for _, s := range blockDeviceTypes() {
		devices, err := filepath.Glob(s)
		if err != nil {
			// Glob can only error on invalid pattern
			panic(fmt.Sprintf("Invalid glob pattern: %s", s))
		}
		for _, device := range devices {
			allDevices = append(allDevices, NewProviderISO(device))
		}
	}
	return allDevices
}

// NewProviderISO returns a new ProviderISO
func NewProviderISO(device string) *ProviderISO {
	return &ProviderISO{device: device}
}

// String returns a string description of ProviderISO
func (p *ProviderISO) String() string {
	return "ISO " + p.device
}

// Probe attemps to mount any of the disks as an ISO volume
// and looks for the config file, it returns true as soon as
// it finds a suitable volume
func (p *ProviderISO) Probe() bool {
	if err := p.mount(); err != nil {
		return false
	}
	if p.configInfo.Mode().IsRegular() && p.configInfo.Size() > 0 {
		return true
	}
	p.cleanup()
	return false
}

// Extract attemps to read config file from a mounted volume, it will
// return an error if it fails to read
func (p *ProviderISO) Extract() ([]byte, error) {
	defer p.cleanup()
	return ioutil.ReadFile(p.configPath)
}

func (p *ProviderISO) mount() error {
	mountPoint, err := ioutil.TempDir("", "mnt")
	if err != nil {
		return err
	}
	p.mountPoint = mountPoint
	// We may need to poll a little for device ready
	err = syscall.Mount(p.device, p.mountPoint, fsType, syscall.MS_RDONLY, "")
	if err != nil {
		p.cleanup()
		return err
	}
	p.configPath = path.Join(p.mountPoint, configFile)
	p.configInfo, err = os.Stat(p.configPath)
	if err != nil {
		p.cleanup()
		return err
	}
	return nil
}

func (p *ProviderISO) cleanup() {
	if p.mountPoint != "" {
		_ = syscall.Unmount(p.mountPoint, 0)
		_ = syscall.Rmdir(p.mountPoint)
	}
}
