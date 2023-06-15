package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/diskfs/go-diskfs"
	log "github.com/sirupsen/logrus"
)

const (
	metadataFile     = "meta-data"
	userdataFile     = "user-data"
	userdataFallback = "config"
	cdromDevs        = "/dev/sr[0-9]*"
	blockDevs        = "/sys/class/block/*"
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
	log.Debugf("cdrom devices to be checked: %v", cdroms)
	// get the devices that match the cloud-init spec
	cidevs := FindCIs()
	log.Debugf("CIDATA devices to be checked: %v", cidevs)
	// merge the two, ensuring that the list is unique
	cdroms = append(cidevs, cdroms...)
	cdroms = uniqueString(cdroms)
	log.Debugf("unique devices to be checked: %v", cdroms)
	var providers []Provider
	for _, device := range cdroms {
		providers = append(providers, NewCDROM(device))
	}
	return providers
}

// FindCIs goes through all known devices. Returns any that are either fat32 or
// iso9660 and have a filesystem label "CIDATA" or "cidata", per the spec
// here https://github.com/canonical/cloud-init/blob/master/doc/rtd/topics/datasources/nocloud.rst
func FindCIs() []string {
	devs, err := filepath.Glob(blockDevs)
	log.Debugf("block devices found: %v", devs)
	if err != nil {
		// Glob can only error on invalid pattern
		panic(fmt.Sprintf("Invalid glob pattern: %s", blockDevs))
	}
	var foundDevices []string
	for _, device := range devs {
		// get the base device name
		dev := filepath.Base(device)
		// ignore loop and ram devices
		if strings.HasPrefix(dev, "loop") || strings.HasPrefix(dev, "ram") {
			log.Debugf("ignoring loop or ram device: %s", dev)
			continue
		}
		dev = fmt.Sprintf("/dev/%s", dev)
		log.Debugf("checking device: %s", dev)
		// open readonly, ignore errors
		disk, err := diskfs.Open(dev, diskfs.WithOpenMode(diskfs.ReadOnly))
		if err != nil {
			log.Debugf("failed to open device read-only: %s: %v", dev, err)
			continue
		}
		disk.DefaultBlocks = true // because this is passed through as a block device, we can get strange blocksize numbers from the OS
		fs, err := disk.GetFilesystem(0)
		if err != nil {
			log.Debugf("failed to get filesystem on partition 0 for device: %s: %v", dev, err)
			continue
		}
		// get the label
		label := strings.TrimSpace(fs.Label())
		log.Debugf("found trimmed filesystem label for device: %s: '%s'", dev, label)
		if label == "cidata" || label == "CIDATA" {
			log.Debugf("adding device: %s", dev)
			foundDevices = append(foundDevices, dev)
		}
	}
	return foundDevices
}

// NewCDROM returns a new ProviderCDROM
func NewCDROM(device string) *ProviderCDROM {
	mountPoint, err := os.MkdirTemp("", "cd")
	p := ProviderCDROM{device, mountPoint, err, []byte{}, []byte{}}
	if err == nil {
		if p.err = p.mount(); p.err == nil {
			// read the userdata - we read the spec file and the fallback, but eventually
			// will remove the fallback
			for _, f := range userdataFiles {
				userdata, err := os.ReadFile(path.Join(p.mountPoint, f))
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
			metadata, err := os.ReadFile(path.Join(p.mountPoint, metadataFile))
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

// uniqueString returns a unique subset of the string slice provided.
func uniqueString(input []string) []string {
	u := make([]string, 0, len(input))
	m := make(map[string]bool)

	for _, val := range input {
		if _, ok := m[val]; !ok {
			m[val] = true
			u = append(u, val)
		}
	}

	return u
}
