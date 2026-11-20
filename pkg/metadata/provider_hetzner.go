package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
)

// hetznerBase is the Hetzner metadata service (a var so tests can override it)
var hetznerBase = "http://169.254.169.254/hetzner/v1/"

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
func (p *ProviderHetzner) Probe(ctx context.Context) bool {
	// Getting the hostname should always work...
	_, err := get(ctx, hetznerBase+"metadata/hostname", nil)
	return err == nil
}

// Extract gets both the Hetzner specific and generic userdata
func (p *ProviderHetzner) Extract(ctx context.Context) ([]byte, error) {
	// Get host name. This must not fail
	hostname, err := get(ctx, hetznerBase+"metadata/hostname", nil)
	if err != nil {
		return nil, fmt.Errorf("Hetzner: %w", err)
	}
	err = os.WriteFile(path.Join(ConfigPath, Hostname), hostname, 0644)
	if err != nil {
		return nil, fmt.Errorf("Hetzner: Failed to write hostname: %s", err)
	}

	// public ipv4
	hetznerMetaGet(ctx, "public-ipv4", "public_ipv4", 0644)

	// instance-id
	hetznerMetaGet(ctx, "instance-id", "instance_id", 0644)

	// ssh
	if err := p.handleSSH(ctx); err != nil {
		log.Printf("Hetzner: Failed to get ssh data: %s", err)
	}

	// Generic userdata
	userData, err := get(ctx, hetznerBase+"userdata", nil)
	if err != nil {
		log.Printf("Hetzner: Failed to get user-data: %s", err)
		// This is not an error
		return nil, nil
	}
	return userData, nil
}

// lookup a value (lookupName) in hetzner metaservice and store in given fileName
func hetznerMetaGet(ctx context.Context, lookupName string, fileName string, fileMode os.FileMode) {
	if lookupValue, err := get(ctx, hetznerBase+"metadata/"+lookupName, nil); err == nil {
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

// SSH keys:
func (p *ProviderHetzner) handleSSH(ctx context.Context) error {
	sshKeysJSON, err := get(ctx, hetznerBase+"metadata/public-keys", nil)
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
