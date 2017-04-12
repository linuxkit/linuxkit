package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
)

const (
	configFile = "config"
)

// ProviderCDROM is the type implementing the Provider interface for CDROMs
// It looks for a file called 'configFile' in the root
type ProviderCDROM struct {
}

// NewCDROM returns a new ProviderCDROM
func NewCDROM() *ProviderCDROM {
	return &ProviderCDROM{}
}

func (p *ProviderCDROM) String() string {
	return "CDROM"
}

// Probe checks if the CD has the right file
func (p *ProviderCDROM) Probe() bool {
	_, err := os.Stat(path.Join(MountPoint, configFile))
	return (!os.IsNotExist(err))
}

// Extract gets both the CDROM specific and generic userdata
func (p *ProviderCDROM) Extract() ([]byte, error) {
	data, err := ioutil.ReadFile(path.Join(MountPoint, configFile))
	if err != nil {
		return nil, fmt.Errorf("CDROM: Error reading file: %s", err)
	}
	return data, nil
}
