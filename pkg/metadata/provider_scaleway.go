package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	instanceIDFile       = "instance_id"
	instanceLocationFile = "instance_location"
	publicIPFile         = "public_ip"
	privateIPFile        = "private_ip"
)

// The Scaleway metadata service and its userdata endpoint (vars so tests can override them)
var (
	scalewayBase         = "http://169.254.42.42/"
	scalewayUserdataAddr = "169.254.42.42:80"
)

// ProviderScaleway is the type implementing the Provider interface for Scaleway
type ProviderScaleway struct {
}

// NewScaleway returns a new ProviderScaleway
func NewScaleway() *ProviderScaleway {
	return &ProviderScaleway{}
}

func (p *ProviderScaleway) String() string {
	return "Scaleway"
}

// Probe checks if we are running on Scaleway
func (p *ProviderScaleway) Probe(ctx context.Context) bool {
	// Getting the conf should always work...
	_, err := get(ctx, scalewayBase+"conf", nil)
	if err != nil {
		log.Printf("Scaleway: %s", err)
		return false
	}

	return true
}

// Extract gets both the Scaleway specific and generic userdata
func (p *ProviderScaleway) Extract(ctx context.Context) ([]byte, error) {
	metadata, err := get(ctx, scalewayBase+"conf", nil)
	if err != nil {
		return nil, fmt.Errorf("Scaleway: Failed to get conf: %s", err)
	}

	hostname, err := p.extractInformation(metadata, "hostname")
	if err != nil {
		return nil, fmt.Errorf("Scaleway: Failed to get hostname: %s", err)
	}

	err = os.WriteFile(path.Join(ConfigPath, Hostname), hostname, 0644)
	if err != nil {
		return nil, fmt.Errorf("Scaleway: Failed to write hostname: %s", err)
	}

	instanceID, err := p.extractInformation(metadata, "id")
	if err != nil {
		return nil, fmt.Errorf("Scaleway: Failed to get instanceID: %s", err)
	}

	err = os.WriteFile(path.Join(ConfigPath, instanceIDFile), instanceID, 0644)
	if err != nil {
		return nil, fmt.Errorf("Scaleway: Failed to write instance_id: %s", err)
	}

	instanceLocation, err := p.extractInformation(metadata, "location_zone_id")
	if err != nil {
		return nil, fmt.Errorf("Scaleway: Failed to get instanceLocation: %s", err)
	}

	err = os.WriteFile(path.Join(ConfigPath, instanceLocationFile), instanceLocation, 0644)
	if err != nil {
		return nil, fmt.Errorf("Scaleway: Failed to write instance_location: %s", err)
	}

	publicIP, err := p.extractInformation(metadata, "public_ip_address")
	if err != nil {
		// not an error
		log.Printf("Scaleway: Failed to get publicIP: %s", err)
	} else {
		err = os.WriteFile(path.Join(ConfigPath, publicIPFile), publicIP, 0644)
		if err != nil {
			return nil, fmt.Errorf("Scaleway: Failed to write public_ip: %s", err)
		}

	}

	privateIP, err := p.extractInformation(metadata, "private_ip")
	if err != nil {
		return nil, fmt.Errorf("Scaleway: Failed to get privateIP: %s", err)
	}

	err = os.WriteFile(path.Join(ConfigPath, privateIPFile), privateIP, 0644)
	if err != nil {
		return nil, fmt.Errorf("Scaleway: Failed to write private_ip: %s", err)
	}

	if err := p.handleSSH(metadata); err != nil {
		log.Printf("Scaleway: Failed to get ssh data: %s", err)
	}

	// Generic userdata
	userData, err := scalewayGetUserdata(ctx)
	if err != nil {
		log.Printf("Scaleway: Failed to get user-data: %s", err)
		// This is not an error
		return nil, nil
	}
	return userData, nil
}

// extractInformation returns the extracted information given as parameter from the metadata
func (p *ProviderScaleway) extractInformation(metadata []byte, information string) ([]byte, error) {
	query := strings.ToUpper(information) + "="
	for _, line := range bytes.Split(metadata, []byte("\n")) {
		if bytes.HasPrefix(line, []byte(query)) {
			return bytes.TrimPrefix(line, []byte(query)), nil
		}
	}
	return []byte(""), fmt.Errorf("No %s found", information)
}

// scalewayGetUserdata returns the userdata of the server, or nil if there is none. Unlike get, it needs
// a source port below 1024, so it speaks HTTP over its own connection; network errors are retried.
func scalewayGetUserdata(ctx context.Context) ([]byte, error) {
	var body []byte
	err := retry(ctx, "GET "+scalewayUserdataAddr+"/user_data/cloud-init", func() error {
		var conn net.Conn
		var err error
		for port := 1; port <= 1024 && conn == nil; port++ {
			d := net.Dialer{LocalAddr: &net.TCPAddr{Port: port}, Timeout: 2 * time.Second}
			conn, err = d.DialContext(ctx, "tcp", scalewayUserdataAddr)
			if err != nil && !errors.Is(err, syscall.EADDRINUSE) && !errors.Is(err, syscall.EACCES) {
				return err
			}
		}
		if conn == nil {
			return fmt.Errorf("not able to find a free port below 1024: %w", err)
		}
		defer conn.Close()
		if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(conn, "GET /user_data/cloud-init HTTP/1.0\r\n\r\n"); err != nil {
			return err
		}
		resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		switch resp.StatusCode {
		case http.StatusOK:
			body, err = io.ReadAll(resp.Body)
			return err
		case http.StatusNotFound:
			return nil
		}
		return &statusError{resp.StatusCode}
	})
	return body, err
}

func (p *ProviderScaleway) handleSSH(metadata []byte) error {
	sshKeysNumberString, err := p.extractInformation(metadata, "ssh_public_keys")
	if err != nil {
		return fmt.Errorf("Failed to get sshKeys: %s", err)
	}
	sshKeysNumber, err := strconv.Atoi(string(sshKeysNumberString))
	if err != nil {
		return fmt.Errorf("Failed to convert sshKeysNumber to int: %s", err)
	}

	rootKeys := ""
	for i := 0; i < sshKeysNumber; i++ {
		sshKey, err := p.extractInformation(metadata, "ssh_public_keys_"+strconv.Itoa(i)+"_key")
		if err != nil {
			return fmt.Errorf("Failed to get ssh_key %d: %s", i, err)
		}

		line := string(bytes.Trim(sshKey, "'"))
		rootKeys = rootKeys + line + "\n"
	}

	if err := os.Mkdir(path.Join(ConfigPath, SSH), 0755); err != nil {
		return fmt.Errorf("Failed to create %s: %s", SSH, err)
	}

	err = os.WriteFile(path.Join(ConfigPath, SSH, "authorized_keys"), []byte(rootKeys), 0600)
	if err != nil {
		return fmt.Errorf("Failed to write ssh keys: %s", err)
	}
	return nil
}
