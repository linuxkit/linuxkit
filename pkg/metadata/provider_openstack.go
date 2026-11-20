package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
)

// ProviderOpenstack is the type implementing the Provider interface for OpenStack
type ProviderOpenstack struct {
}

// NewOpenstack returns a new ProviderOpenstack
func NewOpenstack() *ProviderOpenstack {
	return &ProviderOpenstack{}
}

func (p *ProviderOpenstack) String() string {
	return "openstack"
}

// Probe checks if we are running on OpenStack
func (p *ProviderOpenstack) Probe(ctx context.Context) bool {
	// Getting the hostname should always work...
	_, err := get(ctx, ec2Base+"latest/meta-data/hostname", nil)
	return err == nil
}

// Extract gets both the OpenStack specific and generic userdata
func (p *ProviderOpenstack) Extract(ctx context.Context) ([]byte, error) {
	// Get host name. This must not fail
	hostname, err := get(ctx, ec2Base+"latest/meta-data/hostname", nil)
	if err != nil {
		return nil, fmt.Errorf("OpenStack: %w", err)
	}
	err = os.WriteFile(path.Join(ConfigPath, Hostname), hostname, 0644)
	if err != nil {
		return nil, fmt.Errorf("OpenStack: Failed to write hostname: %s", err)
	}

	// public ipv4
	openstackMetaGet(ctx, "public-ipv4", "public_ipv4", 0644)

	// private ipv4
	openstackMetaGet(ctx, "local-ipv4", "local_ipv4", 0644)

	// availability zone
	openstackMetaGet(ctx, "placement/availability-zone", "availability_zone", 0644)

	// instance type
	openstackMetaGet(ctx, "instance-type", "instance_type", 0644)

	// instance-id
	openstackMetaGet(ctx, "instance-id", "instance_id", 0644)

	// local-hostname
	openstackMetaGet(ctx, "local-hostname", "local_hostname", 0644)

	// ssh
	if err := p.handleSSH(ctx); err != nil {
		log.Printf("OpenStack: Failed to get ssh data: %s", err)
	}

	// Generic userdata
	userData, err := get(ctx, ec2Base+"latest/user-data", nil)
	if err != nil {
		log.Printf("OpenStack: Failed to get user-data: %s", err)
		// This is not an error
		return nil, nil
	}
	return userData, nil
}

// lookup a value (lookupName) in OpenStack's metaservice and store in given fileName
func openstackMetaGet(ctx context.Context, lookupName string, fileName string, fileMode os.FileMode) {
	if lookupValue, err := get(ctx, ec2Base+"latest/meta-data/"+lookupName, nil); err == nil {
		// we got a value from the metadata server, now save to filesystem
		err = os.WriteFile(path.Join(ConfigPath, fileName), lookupValue, fileMode)
		if err != nil {
			// we couldn't save the file for some reason
			log.Printf("OpenStack: Failed to write %s:%s %s", fileName, lookupValue, err)
		}
	} else {
		// we did not get a value back from the metadata server
		log.Printf("OpenStack: Failed to get %s: %s", lookupName, err)
	}
}

// SSH keys:
func (p *ProviderOpenstack) handleSSH(ctx context.Context) error {
	sshKeys, err := get(ctx, ec2Base+"latest/meta-data/public-keys/0/openssh-key", nil)
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
