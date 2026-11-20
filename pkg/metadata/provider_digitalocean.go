package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
)

// digitalOceanBase is the DigitalOcean metadata service (a var so tests can override it)
var digitalOceanBase = "http://169.254.169.254/metadata/v1/"

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
func (p *ProviderDigitalOcean) Probe(ctx context.Context) bool {
	// Getting the index should always work...
	_, err := get(ctx, digitalOceanBase, nil)
	return err == nil
}

// Extract gets both the DigitalOcean specific and generic userdata
func (p *ProviderDigitalOcean) Extract(ctx context.Context) ([]byte, error) {
	// Get host name. This must not fail
	hostname, err := get(ctx, digitalOceanBase+"hostname", nil)
	if err != nil {
		return nil, fmt.Errorf("DigitalOcean: %w", err)
	}
	err = os.WriteFile(path.Join(ConfigPath, Hostname), hostname, 0644)
	if err != nil {
		return nil, fmt.Errorf("DigitalOcean: Failed to write hostname: %s", err)
	}

	// public ipv4
	digitalOceanMetaGet(ctx, "interfaces/public/0/ipv4/address", "public_ipv4", 0644)

	// private ipv4
	digitalOceanMetaGet(ctx, "interfaces/private/0/ipv4/address", "private_ipv4", 0644)

	// region
	digitalOceanMetaGet(ctx, "region", "region", 0644)

	// droplet id
	digitalOceanMetaGet(ctx, "id", "id", 0644)

	// ssh
	if err := p.handleSSH(ctx); err != nil {
		log.Printf("DigitalOcean: Failed to get ssh data: %s", err)
	}

	// Generic userdata
	userData, err := get(ctx, digitalOceanBase+"user-data", nil)
	if err != nil {
		log.Printf("DigitalOcean: Failed to get user-data: %s", err)
		// This is not an error
		return nil, nil
	}
	return userData, nil
}

// lookup a value (lookupName) in DigitalOcean metaservice and store in given fileName
func digitalOceanMetaGet(ctx context.Context, lookupName string, fileName string, fileMode os.FileMode) {
	if lookupValue, err := get(ctx, digitalOceanBase+lookupName, nil); err == nil {
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

// SSH keys:
func (p *ProviderDigitalOcean) handleSSH(ctx context.Context) error {
	sshKeys, err := get(ctx, digitalOceanBase+"public-keys", nil)
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
