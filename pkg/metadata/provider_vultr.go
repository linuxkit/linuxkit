package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"time"
)

const (
	vultrMetaDataURL = "http://169.254.169.254/v1/"
)

// ProviderVultr is the type implementing the Provider interface for Vultr
type ProviderVultr struct {
}

// NewVultr returns a new ProviderVultr
func NewVultr() *ProviderVultr {
	return &ProviderVultr{}
}

func (p *ProviderVultr) String() string {
	return "Vultr"
}

// Probe checks if we are running on Vultr
func (p *ProviderVultr) Probe() bool {
	// Getting the index should always work...
	_, err := vultrGet(vultrMetaDataURL)
	return err == nil
}

// Extract gets both the Vultr specific and generic userdata
func (p *ProviderVultr) Extract() ([]byte, error) {
	// Get host name. This must not fail
	hostname, err := vultrGet(vultrMetaDataURL + "hostname")
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(path.Join(ConfigPath, Hostname), hostname, 0644)
	if err != nil {
		return nil, fmt.Errorf("Vultr: Failed to write hostname: %s", err)
	}

	// public ipv4
	vultrMetaGet("interfaces/0/ipv4/address", "public_ipv4", 0644)

	// private ipv4
	vultrMetaGet("interfaces/1/ipv4/address", "private_ipv4", 0644)

	// private netmask
	vultrMetaGet("interfaces/1/ipv4/netmask", "private_netmask", 0644)

	// region code
	vultrMetaGet("region/regioncode", "region_code", 0644)

	// instance-id
	vultrMetaGet("instanceid", "instance_id", 0644)

	// ssh
	if err := p.handleSSH(); err != nil {
		log.Printf("Vultr: Failed to get ssh data: %s", err)
	}

	return nil, nil
}

// lookup a value (lookupName) in Vultr metaservice and store in given fileName
func vultrMetaGet(lookupName string, fileName string, fileMode os.FileMode) {
	if lookupValue, err := vultrGet(vultrMetaDataURL + lookupName); err == nil {
		// we got a value from the metadata server, now save to filesystem
		err = os.WriteFile(path.Join(ConfigPath, fileName), lookupValue, fileMode)
		if err != nil {
			// we couldn't save the file for some reason
			log.Printf("Vultr: Failed to write %s:%s %s", fileName, lookupValue, err)
		}
	} else {
		// we did not get a value back from the metadata server
		log.Printf("Vultr: Failed to get %s: %s", lookupName, err)
	}
}

// vultrGet requests and extracts the requested URL
func vultrGet(url string) ([]byte, error) {
	var client = &http.Client{
		Timeout: time.Second * 2,
	}

	req, err := http.NewRequest("", url, nil)
	if err != nil {
		return nil, fmt.Errorf("Vultr: http.NewRequest failed: %s", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Vultr: Could not contact metadata service: %s", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Vultr: Status not ok: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Vultr: Failed to read http response: %s", err)
	}
	return body, nil
}

// SSH keys:
func (p *ProviderVultr) handleSSH() error {
	sshKeys, err := vultrGet(vultrMetaDataURL + "public-keys")
	if err != nil {
		return fmt.Errorf("Failed to get sshKeys: %s", err)
	}

	if err := os.Mkdir(path.Join(ConfigPath, SSH), 0755); err != nil {
		return fmt.Errorf("Failed to create %s: %s", SSH, err)
	}

	err = os.WriteFile(path.Join(ConfigPath, SSH, "authorized_keys"), sshKeys, 0600)
	if err != nil {
		return fmt.Errorf("Failed to write ssh keys: %s", err)
	}
	return nil
}
