package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
)

// metaldataBase is the Metaldata service (a var so tests can override it)
var metaldataBase = "http://metaldata/get/"

// ProviderMetaldata is the type implementing the Provider interface for Metaldata
type ProviderMetaldata struct {
}

// NewMetalData returns a new ProviderMetaldata
func NewMetalData() *ProviderMetaldata {
	return &ProviderMetaldata{}
}

func (p *ProviderMetaldata) String() string {
	return "metaldata"
}

// Probe checks if we are running on Metaldata
func (p *ProviderMetaldata) Probe(ctx context.Context) bool {
	log.Println("Metaldata: Probing...")
	// Getting the hostname should always work...
	_, err := get(ctx, metaldataBase+"meta/hostname", nil)
	return err == nil
}

// Extract gets both the Metaldata specific and generic userdata
func (p *ProviderMetaldata) Extract(ctx context.Context) ([]byte, error) {
	// Get host name. This must not fail
	hostname, err := get(ctx, metaldataBase+"meta/hostname", nil)
	if err != nil {
		return nil, fmt.Errorf("Metaldata: %w", err)
	}
	err = os.WriteFile(path.Join(ConfigPath, Hostname), hostname, 0644)
	if err != nil {
		return nil, fmt.Errorf("Metaldata: Failed to write hostname: %s", err)
	}

	// public ipv4
	metaldataMetaGet(ctx, "public-ipv4", "public_ipv4", 0644)

	// private ipv4
	metaldataMetaGet(ctx, "private-ipv4", "private_ipv4", 0644)

	// failure domain
	metaldataMetaGet(ctx, "failure-domain", "failure_domain", 0644)

	// id
	metaldataMetaGet(ctx, "machine-id", "machine_id", 0644)

	// type
	metaldataMetaGet(ctx, "machine-type", "machine_type", 0644)

	// ssh
	if err := p.handleSSH(ctx); err != nil {
		log.Printf("Metaldata: Failed to get ssh data: %s", err)
	}

	// Generic userdata
	userData, err := get(ctx, metaldataBase+"user", nil)
	if err != nil {
		log.Printf("Metaldata: Failed to get user-data: %s", err)
		// This is not an error
		return nil, nil
	}
	return userData, nil
}

// lookup a value (lookupName) in Metaldata metaservice and store in given fileName
func metaldataMetaGet(ctx context.Context, lookupName string, fileName string, fileMode os.FileMode) {
	if lookupValue, err := get(ctx, metaldataBase+"meta/"+lookupName, nil); err == nil {
		// we got a value from the metadata server, now save to filesystem
		err = os.WriteFile(path.Join(ConfigPath, fileName), lookupValue, fileMode)
		if err != nil {
			// we couldn't save the file for some reason
			log.Printf("Metaldata: Failed to write %s:%s %s", fileName, lookupValue, err)
		}
	} else {
		// we did not get a value back from the metadata server
		log.Printf("Metaldata: Failed to get %s: %s", lookupName, err)
	}
}

// SSH keys:
func (p *ProviderMetaldata) handleSSH(ctx context.Context) error {
	sshKeys, err := get(ctx, metaldataBase+"meta/authorized-keys", nil)
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
