package main

import (
	"bufio"
	"bytes"
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
	"time"
)

const (
	scalewayMetadataURL  = "http://169.254.42.42/"
	scalewayUserdataURL  = "169.254.42.42:80"
	instanceIDFile       = "instance_id"
	instanceLocationFile = "instance_location"
	publicIPFile         = "public_ip"
	privateIPFile        = "private_ip"
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
func (p *ProviderScaleway) Probe() bool {
	// Getting the conf should always work...
	_, err := scalewayGet(scalewayMetadataURL + "conf")
	if err != nil {
		log.Printf(err.Error())
		return false
	}

	return true
}

// Extract gets both the Scaleway specific and generic userdata
func (p *ProviderScaleway) Extract() ([]byte, error) {
	metadata, err := scalewayGet(scalewayMetadataURL + "conf")
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
	userData, err := scalewayGetUserdata()
	if err != nil {
		log.Printf("Scaleway: Failed to get user-data: %s", err)
		// This is not an error
		return nil, nil
	}
	return userData, nil
}

// exctractInformation returns the extracted information given as parameter from the metadata
func (p *ProviderScaleway) extractInformation(metadata []byte, information string) ([]byte, error) {
	query := strings.ToUpper(information) + "="
	for _, line := range bytes.Split(metadata, []byte("\n")) {
		if bytes.HasPrefix(line, []byte(query)) {
			return bytes.TrimPrefix(line, []byte(query)), nil
		}
	}
	return []byte(""), fmt.Errorf("No %s found", information)
}

// scalewayGet requests and extracts the requested URL
func scalewayGet(url string) ([]byte, error) {
	var client = &http.Client{
		Timeout: time.Second * 2,
	}

	req, err := http.NewRequest("", url, nil)
	if err != nil {
		return nil, fmt.Errorf("Scaleway: http.NewRequest failed: %s", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Scaleway: Could not contact metadata service: %s", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Scaleway: Status not ok: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Scaleway: Failed to read http response: %s", err)
	}
	return body, nil
}

// scalewayGetUserdata returns the userdata of the server, differs from scalewayGet since the source port has to be below 1024 in order to work
func scalewayGetUserdata() ([]byte, error) {
	server, err := net.ResolveTCPAddr("tcp", scalewayUserdataURL)
	if err != nil {
		return nil, err
	}
	var conn *net.TCPConn
	foundPort := false
	for i := 1; i <= 1024; i++ {
		client, err := net.ResolveTCPAddr("tcp", ":"+strconv.Itoa(i))
		if err != nil {
			return nil, err
		}

		conn, err = net.DialTCP("tcp", client, server)
		if err == nil {
			foundPort = true
			break
		}
	}
	if foundPort == false {
		return nil, errors.New("not able to found a free port below 1024")
	}
	defer conn.Close()
	fmt.Fprintf(conn, "GET /user_data/cloud-init HTTP/1.0\r\n\r\n")

	reader := bufio.NewReader(conn)
	resp, err := http.ReadResponse(reader, nil)
	if err != nil || resp.StatusCode == 404 {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
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
