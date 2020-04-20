package main

import (
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"syscall"
)

const (
	metadataFile     = "meta-data"
	userdataFile     = "user-data"
	userdataFallback = "config"
	cdromDevs        = "/dev/sr[0-9]*"
)

var (
	userdataFiles = []string{userdataFile, userdataFallback}
)

// ProviderCDROM is the type implementing the Provider interface for CDROMs
// It looks for file called 'meta-data', 'user-data' or 'config' in the root
type ProviderCDROM struct {
	device             string
	mountPoint         string
	err                error
	userdata, metadata []byte
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
	p := ProviderCDROM{device, mountPoint, err, []byte{}, []byte{}}
	if err == nil {
		if p.err = p.mount(); p.err == nil {
			// read the userdata - we read the spec file and the fallback, but eventually
			// will remove the fallback
			for _, f := range userdataFiles {
				userdata, err := ioutil.ReadFile(path.Join(p.mountPoint, f))
				// did we find a file?
				if err == nil && userdata != nil {
					p.userdata = userdata
					break
				}
			}
			if p.userdata == nil {
				p.err = fmt.Errorf("no userdata file found at any of %v", userdataFiles)
			}
			// read the metadata
			metadata, err := ioutil.ReadFile(path.Join(p.mountPoint, metadataFile))
			// did we find a file?
			if err == nil && metadata != nil {
				p.metadata = metadata
			}
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
	return len(p.userdata) != 0
}

// Extract gets both the CDROM specific and generic userdata
func (p *ProviderCDROM) Extract() ([]byte, error) {
	return p.userdata, p.err
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
