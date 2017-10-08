package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"time"
)

// ProviderAWS is the type implementing the Provider interface for AWS
type ProviderAWS struct {
}

// NewAWS returns a new ProviderAWS
func NewAWS() *ProviderAWS {
	return &ProviderAWS{}
}

func (p *ProviderAWS) String() string {
	return "AWS"
}

// Probe checks if we are running on AWS
func (p *ProviderAWS) Probe() bool {
	// Getting the hostname should always work...
	_, err := awsGet(metaDataURL + "hostname")
	return (err == nil)
}

// Extract gets both the AWS specific and generic userdata
func (p *ProviderAWS) Extract() ([]byte, error) {
	// Get host name. This must not fail
	hostname, err := awsGet(metaDataURL + "hostname")
	if err != nil {
		return nil, err
	}
	err = ioutil.WriteFile(path.Join(ConfigPath, Hostname), hostname, 0644)
	if err != nil {
		return nil, fmt.Errorf("AWS: Failed to write hostname: %s", err)
	}

	// public ipv4
	awsMetaGet("public-ipv4", "public_ipv4", 0644)

	// private ipv4
	awsMetaGet("local-ipv4", "local_ipv4", 0644)

	// availability zone
	awsMetaGet("placement/availability-zone", "availability_zone", 0644)

	// instance type
	awsMetaGet("instance-type", "instance_type", 0644)

	// instance-id
	awsMetaGet("instance-id", "instance_id", 0644)

	// local-hostname
	awsMetaGet("local-hostname", "local_hostname", 0644)

	// ssh
	if err := p.handleSSH(); err != nil {
		log.Printf("AWS: Failed to get ssh data: %s", err)
	}

	// Generic userdata
	userData, err := awsGet(userDataURL)
	if err != nil {
		log.Printf("AWS: Failed to get user-data: %s", err)
		// This is not an error
		return nil, nil
	}
	return userData, nil
}

// lookup a value (lookupName) in aws metaservice and store in given fileName
func awsMetaGet(lookupName string, fileName string, fileMode os.FileMode) {
	if lookupValue, err := awsGet(metaDataURL + lookupName); err == nil {
		// we got a value from the metadata server, now save to filesystem
		err = ioutil.WriteFile(path.Join(ConfigPath, fileName), lookupValue, fileMode)
		if err != nil {
			// we couldn't save the file for some reason
			log.Printf("AWS: Failed to write %s:%s %s", fileName, lookupValue, err)
		}
	} else {
		// we did not get a value back from the metadata server
		log.Printf("AWS: Failed to get %s: %s", lookupName, err)
	}
}

// awsGet requests and extracts the requested URL
func awsGet(url string) ([]byte, error) {
	var client = &http.Client{
		Timeout: time.Second * 2,
	}

	req, err := http.NewRequest("", url, nil)
	if err != nil {
		return nil, fmt.Errorf("AWS: http.NewRequest failed: %s", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("AWS: Could not contact metadata service: %s", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("AWS: Status not ok: %d", resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("AWS: Failed to read http response: %s", err)
	}
	return body, nil
}

// SSH keys:
func (p *ProviderAWS) handleSSH() error {
	sshKeys, err := awsGet(metaDataURL + "public-keys/0/openssh-key")
	if err != nil {
		return fmt.Errorf("Failed to get sshKeys: %s", err)
	}

	if err := os.Mkdir(path.Join(ConfigPath, SSH), 0755); err != nil {
		return fmt.Errorf("Failed to create %s: %s", SSH, err)
	}

	err = ioutil.WriteFile(path.Join(ConfigPath, SSH, "authorized_keys"), sshKeys, 0600)
	if err != nil {
		return fmt.Errorf("Failed to write ssh keys: %s", err)
	}
	return nil
}
