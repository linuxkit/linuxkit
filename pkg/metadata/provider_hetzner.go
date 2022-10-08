package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"time"
)

// ProviderHetzner is the type implementing the Provider interface for Hetzner
type ProviderHetzner struct {
}

// NewHetzner returns a new ProviderHetzner
func NewHetzner() *ProviderHetzner {
	return &ProviderHetzner{}
}

func (p *ProviderHetzner) String() string {
	return "Hetzner"
}

// Probe checks if we are running on Hetzner
func (p *ProviderHetzner) Probe() bool {
	// Getting the hostname should always work...
	_, err := hetznerGet(metaDataURL + "hostname")
	return err == nil
}

// Extract gets both the Hetzner specific and generic userdata
func (p *ProviderHetzner) Extract() ([]byte, error) {
	// Get host name. This must not fail
	hostname, err := hetznerGet(metaDataURL + "hostname")
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(path.Join(ConfigPath, Hostname), hostname, 0644)
	if err != nil {
		return nil, fmt.Errorf("Hetzner: Failed to write hostname: %s", err)
	}

	// public ipv4
	hetznerMetaGet("public-ipv4", "public_ipv4", 0644)

	// private ipv4
	hetznerMetaGet("local-ipv4", "local_ipv4", 0644)

	// instance-id
	hetznerMetaGet("instance-id", "instance_id", 0644)

	// // local-hostname
	// hetznerMetaGet("local-hostname", "local_hostname", 0644)

	// ssh
	if err := p.handleSSH(); err != nil {
		log.Printf("Hetzner: Failed to get ssh data: %s", err)
	}

	// Generic userdata
	userData, err := hetznerGet(userDataURL)
	if err != nil {
		log.Printf("Hetzner: Failed to get user-data: %s", err)
		// This is not an error
		return nil, nil
	}
	return userData, nil
}

// lookup a value (lookupName) in hetzner metaservice and store in given fileName
func hetznerMetaGet(lookupName string, fileName string, fileMode os.FileMode) {
	if lookupValue, err := hetznerGet(metaDataURL + lookupName); err == nil {
		// we got a value from the metadata server, now save to filesystem
		err = os.WriteFile(path.Join(ConfigPath, fileName), lookupValue, fileMode)
		if err != nil {
			// we couldn't save the file for some reason
			log.Printf("Hetzner: Failed to write %s:%s %s", fileName, lookupValue, err)
		}
	} else {
		// we did not get a value back from the metadata server
		log.Printf("Hetzner: Failed to get %s: %s", lookupName, err)
	}
}

// hetznerGet requests and extracts the requested URL
func hetznerGet(url string) ([]byte, error) {
	var client = &http.Client{
		Timeout: time.Second * 2,
	}

	req, err := http.NewRequest("", url, nil)
	if err != nil {
		return nil, fmt.Errorf("Hetzner: http.NewRequest failed: %s", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Hetzner: Could not contact metadata service: %s", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Hetzner: Status not ok: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Hetzner: Failed to read http response: %s", err)
	}
	return body, nil
}

// SSH keys:
func (p *ProviderHetzner) handleSSH() error {
	sshKeysJSON, err := hetznerGet(metaDataURL + "public-keys")
	if err != nil {
		return fmt.Errorf("Failed to get sshKeys: %s", err)
	}

	var sshKeys []string
	err = json.Unmarshal(sshKeysJSON, &sshKeys)
	if err != nil {
		return fmt.Errorf("Failed to get sshKeys: %s", err)
	}

	if err := os.Mkdir(path.Join(ConfigPath, SSH), 0755); err != nil {
		return fmt.Errorf("Failed to create %s: %s", SSH, err)
	}

	fileHandle, _ := os.OpenFile(path.Join(ConfigPath, SSH, "authorized_keys"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	defer fileHandle.Close()

	for _, sshKey := range sshKeys {
		_, err = fileHandle.WriteString(sshKey + "\n")
		if err != nil {
			return fmt.Errorf("Failed to write ssh keys: %s", err)
		}
	}

	return nil
}
