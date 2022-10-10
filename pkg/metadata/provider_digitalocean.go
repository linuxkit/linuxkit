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
	digitalOceanMetaDataURL = "http://169.254.169.254/metadata/v1/"
)

// ProviderDigitalOcean is the type implementing the Provider interface for DigitalOcean
type ProviderDigitalOcean struct {
}

// NewDigitalOcean returns a new ProviderDigitalOcean
func NewDigitalOcean() *ProviderDigitalOcean {
	return &ProviderDigitalOcean{}
}

func (p *ProviderDigitalOcean) String() string {
	return "DigitalOcean"
}

// Probe checks if we are running on DigitalOcean
func (p *ProviderDigitalOcean) Probe() bool {
	// Getting the index should always work...
	_, err := digitalOceanGet(digitalOceanMetaDataURL)
	return err == nil
}

// Extract gets both the DigitalOcean specific and generic userdata
func (p *ProviderDigitalOcean) Extract() ([]byte, error) {
	// Get host name. This must not fail
	hostname, err := digitalOceanGet(digitalOceanMetaDataURL + "hostname")
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(path.Join(ConfigPath, Hostname), hostname, 0644)
	if err != nil {
		return nil, fmt.Errorf("DigitalOcean: Failed to write hostname: %s", err)
	}

	// public ipv4
	digitalOceanMetaGet("interfaces/public/0/ipv4/address", "public_ipv4", 0644)

	// private ipv4
	digitalOceanMetaGet("interfaces/private/0/ipv4/address", "private_ipv4", 0644)

	// region
	digitalOceanMetaGet("region", "region", 0644)

	// droplet id
	digitalOceanMetaGet("id", "id", 0644)

	// ssh
	if err := p.handleSSH(); err != nil {
		log.Printf("DigitalOcean: Failed to get ssh data: %s", err)
	}

	// Generic userdata
	userData, err := digitalOceanGet(digitalOceanMetaDataURL + "user-data")
	if err != nil {
		log.Printf("DigitalOcean: Failed to get user-data: %s", err)
		// This is not an error
		return nil, nil
	}
	return userData, nil
}

// lookup a value (lookupName) in DigitalOcean metaservice and store in given fileName
func digitalOceanMetaGet(lookupName string, fileName string, fileMode os.FileMode) {
	if lookupValue, err := digitalOceanGet(digitalOceanMetaDataURL + lookupName); err == nil {
		// we got a value from the metadata server, now save to filesystem
		err = os.WriteFile(path.Join(ConfigPath, fileName), lookupValue, fileMode)
		if err != nil {
			// we couldn't save the file for some reason
			log.Printf("DigitalOcean: Failed to write %s:%s %s", fileName, lookupValue, err)
		}
	} else {
		// we did not get a value back from the metadata server
		log.Printf("DigitalOcean: Failed to get %s: %s", lookupName, err)
	}
}

// digitalOceanGet requests and extracts the requested URL
func digitalOceanGet(url string) ([]byte, error) {
	var client = &http.Client{
		Timeout: time.Second * 2,
	}

	req, err := http.NewRequest("", url, nil)
	if err != nil {
		return nil, fmt.Errorf("DigitalOcean: http.NewRequest failed: %s", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("DigitalOcean: Could not contact metadata service: %s", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("DigitalOcean: Status not ok: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("DigitalOcean: Failed to read http response: %s", err)
	}
	return body, nil
}

// SSH keys:
func (p *ProviderDigitalOcean) handleSSH() error {
	sshKeys, err := digitalOceanGet(digitalOceanMetaDataURL + "public-keys")
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
