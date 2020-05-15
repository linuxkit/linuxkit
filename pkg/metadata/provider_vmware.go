//go:build linux && 386 && amd64

package main

import (
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"path"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/vmware/vmw-guestinfo/rpcvmx"
	"github.com/vmware/vmw-guestinfo/vmcheck"
)

const (
	guestMetaData = "guestinfo.metadata"

	guestUserData = "guestinfo.userdata"

	guestVendorData = "guestinfo.vendordata"
)

// ProviderVMware implements the Provider interface for VMware guestinfo api
type ProviderVMware struct{}

// NewVMware returns a new VMware Provider
func NewVMware() *ProviderVMware {
	return &ProviderVMware{}
}

// String returns provider name
func (p *ProviderVMware) String() string {
	return "VMWARE"
}

// Probe checks if we are running on VMware and either userdata or metadata is set
func (p *ProviderVMware) Probe() bool {
	isVM, err := vmcheck.IsVirtualWorld()
	if err != nil || !isVM {
		return false
	}

	md, merr := vmwareGet(guestMetaData)
	ud, uerr := vmwareGet(guestUserData)

	return ((merr == nil) && len(md) > 1 && string(md) != "---") || ((uerr == nil) && len(ud) > 1 && string(ud) != "---")
}

// Extract gets the host specific metadata, generic userdata and if set vendordata
// This function returns error if it fails to write metadata or vendordata to disk
func (p *ProviderVMware) Extract() ([]byte, error) {
	// Get vendor data, if empty do not fail
	vendorData, err := vmwareGet(guestVendorData)
	if err != nil {
		log.Debugf("VMWare: Failed to get vendordata: %v", err)
	} else {
		err = ioutil.WriteFile(path.Join(ConfigPath, "vendordata"), vendorData, 0644)
		if err != nil {
			log.Debugf("VMWare: Failed to write vendordata: %v", err)
		}
	}

	// Get metadata
	metaData, err := vmwareGet(guestMetaData)
	if err != nil {
		log.Printf("VMWare: Failed to get metadata: %v", err)
	} else {
		err = ioutil.WriteFile(path.Join(ConfigPath, "metadata"), metaData, 0644)
		if err != nil {
			return nil, fmt.Errorf("VMWare: Failed to write metadata: %s", err)
		}
	}

	// Get userdata
	userData, err := vmwareGet(guestUserData)
	if err != nil {
		log.Printf("VMware: Failed to get userdata: %v", err)
		// This is not an error
		return nil, nil
	}

	return userData, nil
}

// vmwareGet gets and extracts the guestinfo data
func vmwareGet(name string) ([]byte, error) {
	config := rpcvmx.NewConfig()

	// get the gusest info value
	out, err := config.String(name, "")
	if err != nil {
		log.Debugf("Getting guest info %s failed: error %s", name, err)
		return nil, err
	}

	enc, err := config.String(name+".encoding", "")
	if err != nil {
		log.Debugf("Getting guest info %s.encoding failed: error %s", name, err)
		return nil, err
	}

	switch strings.TrimSuffix(enc, "\n") {
	case " ":
		return []byte(strings.TrimSuffix(out, "\n")), nil
	case "base64":
		r := base64.NewDecoder(base64.StdEncoding, strings.NewReader(out))

		dst, err := ioutil.ReadAll(r)
		if err != nil {
			log.Debugf("Decoding base64 of '%s' failed %v", name, err)
			return nil, err
		}

		return dst, nil
	case "gzip+base64":
		r := base64.NewDecoder(base64.StdEncoding, strings.NewReader(out))

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
