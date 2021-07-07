package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	guestMetaData = "guestinfo.metadata"

	guestUserData = "guestinfo.userdata"
)

// ProviderVMware is the type implementing the Provider interface for VMware
type ProviderVMware struct {
	cmd string
}

// NewVMware returns a new ProviderVMware
func NewVMware() *ProviderVMware {
	return &ProviderVMware{}
}

func (p *ProviderVMware) String() string {
	return "VMWARE"
}

// Probe checks if we are running on VMware
func (p *ProviderVMware) Probe() bool {
	c, err := exec.LookPath("vmware-rpctool")
	if err != nil {
		return false
	}

	p.cmd = c

	b, err := p.vmwareGet(guestUserData)
	return (err == nil) && len(b) > 0 && string(b) != " " && string(b) != "---"
}

// Extract gets both the AWS specific and generic userdata
func (p *ProviderVMware) Extract() ([]byte, error) {
	// Get host name. This must not fail
	metaData, err := p.vmwareGet(guestMetaData)
	if err != nil {
		return nil, err
	}

	err = ioutil.WriteFile(path.Join(ConfigPath, "metadata"), metaData, 0644)
	if err != nil {
		return nil, fmt.Errorf("VMWare: Failed to write metadata: %s", err)
	}

	// Generic userdata
	userData, err := p.vmwareGet(guestUserData)
	if err != nil {
		log.Printf("VMware: Failed to get user-data: %s", err)
		// This is not an error
		return nil, nil
	}

	return userData, nil
}

// vmwareGet gets and extracts the guest data
func (p *ProviderVMware) vmwareGet(name string) ([]byte, error) {
	cmdArg := func(n string) string {
		return fmt.Sprintf("info-get %s", n)
	}
	// get the gusest info value
	out, err := exec.Command(p.cmd, cmdArg(name)).Output()
	if err != nil {
		eErr := err.(*exec.ExitError)
		log.Debugf("Getting guest info %s failed: error %s", cmdArg(name), string(eErr.Stderr))
		return nil, err
	}

	enc, err := exec.Command(p.cmd, cmdArg(name+".encoding")).Output()
	if err != nil {
		eErr := err.(*exec.ExitError)
		log.Debugf("Getting guest info %s.encoding failed: error %s", name, string(eErr.Stderr))
		return nil, err
	}

	switch strings.TrimSuffix(string(enc), "\n") {
	case " ":
		return bytes.TrimSuffix(out, []byte("\n")), nil
	case "base64":
		r := base64.NewDecoder(base64.StdEncoding, bytes.NewBuffer(out))

		dst, err := ioutil.ReadAll(r)
		if err != nil {
			log.Debugf("Decoding base64 of '%s' failed %v", name, err)
			return nil, err
		}

		return dst, nil
	case "gzip+base64":
		r := base64.NewDecoder(base64.StdEncoding, bytes.NewBuffer(out))

		zr, err := gzip.NewReader(r)
		if err != nil {
			log.Debugf("New gzip reader from '%s' failed %v", name, err)
			return nil, err
		}

		dst, err := ioutil.ReadAll(zr)
		if err != nil {
			log.Debugf("Read '%s' failed %v", name, err)
			return nil, err
		}

		return dst, nil
	default:
		return nil, fmt.Errorf("Unknown encoding %s", string(enc))
	}
}
