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
	cdromDevs  = "/dev/sr[0-9]*"
)

// ProviderCDROM is the type implementing the Provider interface for CDROMs
// It looks for a file called 'configFile' in the root
type ProviderCDROM struct {
	device     string
	mountPoint string
	err        error
	data       []byte
}

// ListCDROMs lists all the cdroms in the system
func ListCDROMs() []Provider {
	cdroms, err := filepath.Glob(cdromDevs)
	if err != nil {
		// Glob can only error on invalid pattern
		panic(fmt.Sprintf("Invalid glob pattern: %s", cdromDevs))
	}
	providers := []Provider{}
	for _, device := range cdroms {
		providers = append(providers, NewCDROM(device))
	}
	return providers
}

// NewCDROM returns a new ProviderCDROM
func NewCDROM(device string) *ProviderCDROM {
	mountPoint, err := ioutil.TempDir("", "cd")
	p := ProviderCDROM{device, mountPoint, err, []byte{}}
	if err == nil {
		if p.err = p.mount(); p.err == nil {
			p.data, p.err = ioutil.ReadFile(path.Join(p.mountPoint, configFile))
			p.unmount()
		}
	}
	return &p
}

func (p *ProviderCDROM) String() string {
	return "CDROM " + p.device
}

// Probe checks if the CD has the right file
func (p *ProviderCDROM) Probe() bool {
	return len(p.data) != 0
}

// Extract gets both the CDROM specific and generic userdata
func (p *ProviderCDROM) Extract() ([]byte, error) {
	return p.data, p.err
}

// mount mounts a CDROM/DVD device under mountPoint
func (p *ProviderCDROM) mount() error {
	// We may need to poll a little for device ready
	return syscall.Mount(p.device, p.mountPoint, "iso9660", syscall.MS_RDONLY, "")
}

// unmount removes the mount
func (p *ProviderCDROM) unmount() {
	_ = syscall.Unmount(p.mountPoint, 0)
}
