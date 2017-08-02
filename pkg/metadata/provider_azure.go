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

const (
	azMetaDataURL     = "http://169.254.169.254/metadata/instance/"
	azMetadataVersion = "2017-04-02"
	azUserDataLoc     = "/var/lib/waagent/ovf-env.xml"
)

// ProviderAzure is the type implementing the Provider interface for Azure
type ProviderAzure struct {
}

// NewAzure returns a new ProviderAzure
func NewAzure() *ProviderAzure {
	return &ProviderAzure{}
}

func (p *ProviderAzure) String() string {
	return "Azure"
}

// Probe checks if we are running on Azure
func (p *ProviderAzure) Probe() bool {
	// Getting the hostname should always work...
	_, err := azGet(azMetaDataURL + "compute/name")
	return (err == nil)
}

// Extract gets both the Azure specific and generic userdata
func (p *ProviderAzure) Extract() ([]byte, error) {
	// Get host name. This must not fail
	hostname, err := azGet(azMetaDataURL + "compute/name")
	if err != nil {
		return nil, err
	}
	err = ioutil.WriteFile(path.Join(ConfigPath, Hostname), hostname, 0644)
	if err != nil {
		return nil, fmt.Errorf("Azure: Failed to write hostname: %s", err)
	}

	// public ipv4
	azMetaGet("network/interface/0/ipv4/ipAddress/0/publicIpAddress", "public_ipv4", 0644)

	// private ipv4
	azMetaGet("network/interface/0/ipv4/ipAddress/0/privateIpAddress", "local_ipv4", 0644)

	// availability zone
	azMetaGet("compute/location", "availability_zone", 0644)

	// instance type
	azMetaGet("compute/vmSize", "instance_type", 0644)

	// instance-id
	azMetaGet("compute/vmId", "instance_id", 0644)

	// local-hostname
	azMetaGet("compute/name", "local_hostname", 0644)

	// ssh is not available in Azure

	// Generic userdata, available as XML data
	userData, err := getUserData()
	if err != nil {
		log.Printf("Azure: Failed to get user-data: %s", err)
		// This is not an error
		return nil, nil
	}
	return userData, nil
}

func getUserData() ([]byte, error) {
	return ioutil.ReadFile(azUserDataLoc)
}

// lookup a value (lookupName) in azure metaservice and store in given fileName
func azMetaGet(lookupName string, fileName string, fileMode os.FileMode) {
	if lookupValue, err := azGet(azMetaDataURL + lookupName); err == nil {
		// we got a value from the metadata server, now save to filesystem
		err = ioutil.WriteFile(path.Join(ConfigPath, fileName), lookupValue, fileMode)
		if err != nil {
			// we couldn't save the file for some reason
			log.Printf("Azure: Failed to write %s:%s %s", fileName, lookupValue, err)
		}
	} else {
		// we did not get a value back from the metadata server
		log.Printf("Azure: Failed to get %s: %s", lookupName, err)
	}
}

// azGet requests and extracts the requested URL
func azGet(url string) ([]byte, error) {
	var client = &http.Client{
		Timeout: time.Second * 2,
	}

	req, err := http.NewRequest("", url+"?api-version="+azMetadataVersion+"&format=text", nil)
	req.Header.Add("Metadata", "true")
	if err != nil {
		return nil, fmt.Errorf("Azure: http.NewRequest failed: %s", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Azure: Could not contact metadata service: %s", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Azure: Status not ok: %d", resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Azure: Failed to read http response: %s", err)
	}
	return body, nil
}
