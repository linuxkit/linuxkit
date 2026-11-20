package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
)

var (
	// gcpBase is the GCP metadata service (a var so tests can override it)
	gcpBase = "http://metadata.google.internal/computeMetadata/v1/"
	// gcpHeader must be on every request, or GCP answers 403
	gcpHeader = http.Header{"Metadata-Flavor": {"Google"}}
)

// ProviderGCP is the type implementing the Provider interface for GCP
type ProviderGCP struct {
}

// NewGCP returns a new ProviderGCP
func NewGCP() *ProviderGCP {
	return &ProviderGCP{}
}

func (p *ProviderGCP) String() string {
	return "GCP"
}

// Probe checks if we are running on GCP
func (p *ProviderGCP) Probe(ctx context.Context) bool {
	// Getting the hostname should always work...
	_, err := get(ctx, gcpBase+"instance/hostname", gcpHeader)
	return err == nil
}

// Extract gets both the GCP specific and generic userdata
func (p *ProviderGCP) Extract(ctx context.Context) ([]byte, error) {
	// Get host name. This must not fail
	hostname, err := get(ctx, gcpBase+"instance/hostname", gcpHeader)
	if err != nil {
		return nil, fmt.Errorf("GCP: %w", err)
	}
	// GCP serves the FQDN, often longer than the kernel's 64 bytes; use the short name
	short, _, _ := strings.Cut(strings.TrimSpace(string(hostname)), ".")
	err = os.WriteFile(path.Join(ConfigPath, Hostname), []byte(short), 0644)
	if err != nil {
		return nil, fmt.Errorf("GCP: Failed to write hostname: %s", err)
	}

	if err := p.handleSSH(ctx); err != nil {
		log.Printf("GCP: Failed to get ssh data: %s", err)
	}

	// Generic userdata
	userData, err := get(ctx, gcpBase+"instance/attributes/user-data", gcpHeader)
	if err != nil {
		log.Printf("GCP: Failed to get user-data: %s", err)
		// This is not an error
		return nil, nil
	}
	return userData, nil
}

// SSH keys:
// TODO also retrieve the instance keys and respect block
//
//	project keys see:
//	https://cloud.google.com/compute/docs/instances/ssh-keys
//
// The keys have usernames attached, but as a simplification
// we are going to add them all to one root file
// TODO split them into individual user files and make the ssh
//
//	container construct those users
func (p *ProviderGCP) handleSSH(ctx context.Context) error {
	sshKeys, err := get(ctx, gcpBase+"project/attributes/ssh-keys", gcpHeader)
	if err != nil {
		return fmt.Errorf("Failed to get sshKeys: %s", err)
	}

	if _, err := os.Stat(path.Join(ConfigPath, SSH)); os.IsNotExist(err) {
		if err := os.Mkdir(path.Join(ConfigPath, SSH), 0755); err != nil {
			return fmt.Errorf("Failed to create %s: %s", SSH, err)
		}
	}

	rootKeys := ""
	for _, line := range strings.Split(string(sshKeys), "\n") {
		parts := strings.SplitN(line, ":", 2)
		// ignoring username for now
		if len(parts) == 2 {
			rootKeys = rootKeys + parts[1] + "\n"
		}
	}
	err = os.WriteFile(path.Join(ConfigPath, SSH, "authorized_keys"), []byte(rootKeys), 0600)
	if err != nil {
		return fmt.Errorf("Failed to write ssh keys: %s", err)
	}
	return nil
}
