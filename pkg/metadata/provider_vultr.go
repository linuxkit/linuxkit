package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
)

// vultrBase is the Vultr metadata service (a var so tests can override it)
var vultrBase = "http://169.254.169.254/v1/"

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
func (p *ProviderVultr) Probe(ctx context.Context) bool {
	// Getting the index should always work...
	_, err := get(ctx, vultrBase, nil)
	return err == nil
}

// Extract gets both the Vultr specific and generic userdata
func (p *ProviderVultr) Extract(ctx context.Context) ([]byte, error) {
	// Get host name. This must not fail
	hostname, err := get(ctx, vultrBase+"hostname", nil)
	if err != nil {
		return nil, fmt.Errorf("Vultr: %w", err)
	}
	err = os.WriteFile(path.Join(ConfigPath, Hostname), hostname, 0644)
	if err != nil {
		return nil, fmt.Errorf("Vultr: Failed to write hostname: %s", err)
	}

	// public ipv4
	vultrMetaGet(ctx, "interfaces/0/ipv4/address", "public_ipv4", 0644)

	// private ipv4
	vultrMetaGet(ctx, "interfaces/1/ipv4/address", "private_ipv4", 0644)

	// private netmask
	vultrMetaGet(ctx, "interfaces/1/ipv4/netmask", "private_netmask", 0644)

	// region code
	vultrMetaGet(ctx, "region/regioncode", "region_code", 0644)

	// instance-id
	vultrMetaGet(ctx, "instanceid", "instance_id", 0644)

	// ssh
	if err := p.handleSSH(ctx); err != nil {
		log.Printf("Vultr: Failed to get ssh data: %s", err)
	}

	return nil, nil
}

// lookup a value (lookupName) in Vultr metaservice and store in given fileName
func vultrMetaGet(ctx context.Context, lookupName string, fileName string, fileMode os.FileMode) {
	if lookupValue, err := get(ctx, vultrBase+lookupName, nil); err == nil {
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

// SSH keys:
func (p *ProviderVultr) handleSSH(ctx context.Context) error {
	sshKeys, err := get(ctx, vultrBase+"public-keys", nil)
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
