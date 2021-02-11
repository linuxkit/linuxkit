package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// ProviderAzureIMDS reads from Azure's Instance Metadata Service (IMDS) API.
type ProviderAzureIMDS struct {
	client      *http.Client
	providerOVF *ProviderAzureOVF
}

// NewAzureIMDS factory
func NewAzureIMDS() *ProviderAzureIMDS {
	client := &http.Client{
		Timeout: time.Second * 2,
	}
	return &ProviderAzureIMDS{
		client:      client,
		providerOVF: NewAzureOVF(client),
	}
}

func (p *ProviderAzureIMDS) String() string {
	return "Azure-IMDS"
}

// Probe checks if Azure IMDS API is available
func (p *ProviderAzureIMDS) Probe() bool {
	// "Poll" VM Unique ID
	// See: https://azure.microsoft.com/en-us/blog/accessing-and-using-azure-vm-unique-id/
	if _, err := p.imdsGet("compute/vmId"); err != nil {
		log.Debugf("%s: Probe failed: %s", p.String(), err)
		return false
	}
	// Probe fallback provider
	return p.providerOVF.Probe()
}

// Extract user data via Azure IMDS.
func (p *ProviderAzureIMDS) Extract() ([]byte, error) {
	if err := p.saveHostname(); err != nil {
		return nil, fmt.Errorf("%s: %s", p.String(), err)
	}

	if err := p.saveSSHKeys(); err != nil {
		log.Warnf("%s: Saving SSH keys failed: %s", p.String(), err)
	}

	p.imdsSave("network/interface/0/ipv4/ipAddress/0/publicIpAddress")
	p.imdsSave("network/interface/0/ipv4/ipAddress/0/privateIpAddress")
	p.imdsSave("compute/zone")
	p.imdsSave("compute/vmId")

	userData, err := p.getUserData()
	if err != nil {
		return nil, fmt.Errorf("%s: %s", p.String(), err)
	}
	return userData, nil
}

func (p *ProviderAzureIMDS) saveHostname() error {
	hostname, err := p.imdsGet("compute/name")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path.Join(ConfigPath, Hostname), hostname, 0644)
	if err != nil {
		return fmt.Errorf("%s: Failed to write hostname: %s", p.String(), err)
	}
	log.Debugf("%s: Saved hostname: %s", p.String(), string(hostname))
	return nil
}

func (p *ProviderAzureIMDS) saveSSHKeys() error {
	// TODO support multiple keys
	sshKey, err := p.imdsGet("compute/publicKeys/0/keyData")
	if err != nil {
		return fmt.Errorf("Getting SSH key failed: %s", err)
	}
	if err := os.Mkdir(path.Join(ConfigPath, SSH), 0755); err != nil {
		return fmt.Errorf("Creating directory %s failed: %s", SSH, err)
	}
	err = ioutil.WriteFile(path.Join(ConfigPath, SSH, "authorized_keys"), sshKey, 0600)
	if err != nil {
		return fmt.Errorf("Writing SSH key failed: %s", err)
	}
	log.Debugf("%s: Saved authorized_keys", p.String())
	return nil
}

// Get resource value from IMDS and write to file in ConfigPath
func (p *ProviderAzureIMDS) imdsSave(resourceName string) {
	if value, err := p.imdsGet(resourceName); err == nil {
		fileName := strings.Replace(resourceName, "/", "_", -1)
		err = ioutil.WriteFile(path.Join(ConfigPath, fileName), value, 0644)
		if err != nil {
			log.Warnf("%s: Failed to write file %s:%s %s", p.String(), fileName, value, err)
		}
		log.Debugf("%s: Saved resource %s: %s", p.String(), resourceName, string(value))
	} else {
		log.Warnf("%s: Failed to get resource %s: %s", p.String(), resourceName, err)
	}
}

// Get IMDS resource value
func (p *ProviderAzureIMDS) imdsGet(resourceName string) ([]byte, error) {
	req, err := http.NewRequest("GET", imdsURL(resourceName), nil)
	if err != nil {
		return nil, fmt.Errorf("http.NewRequest failed: %s", err)
	}
	req.Header.Set("Metadata", "true")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("IMDS unavailable: %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("IMDS returned status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Reading HTTP response failed: %s", err)
	}

	return body, nil
}

// Build Azure Instance Metadata Service (IMDS) URL
// For available nodes, see: https://docs.microsoft.com/en-us/azure/virtual-machines/windows/instance-metadata-service
func imdsURL(node string) string {
	const (
		baseURL = "http://169.254.169.254/metadata/instance"
		// TODO Version 2020-10-01 might not yet be available in every region
		//apiVersion = "2020-10-01"
		apiVersion = "2020-09-01"
		// For leaf nodes in /metadata/instance, the format=json doesn't work.
		// For these queries, format=text needs to be explicitly specified
		// because the default format is JSON.
		params = "?api-version=" + apiVersion + "&format=text"
	)
	if len(node) > 0 {
		return baseURL + "/" + node + params
	}
	return baseURL + params
}

func (p *ProviderAzureIMDS) getUserData() ([]byte, error) {
	userData, err := p.imdsGet("compute/customData")
	if err != nil {
		log.Errorf("Failed to get user data: %s", err)
		return nil, err
	}

	// TODO
	// Fallback OVF provider will report ready
	// defer ReportReady(p.client)

	if len(userData) > 0 { // Always false
		log.Warnf("%s: Unexpectedly received user data: \n%s", p.String(), string(userData))
		// TODO
		// Getting user data via IMDS is disabled. See blocking upstream issue:
		//  * https://github.com/MicrosoftDocs/azure-docs/issues/64154
		//  * https://github.com/MicrosoftDocs/azure-docs/issues/30370 (OP)
		// return userData, nil
	}

	// log.Debugf("%s: user data is empty", p.String())
	// return nil, nil

	// Fallback
	log.Debugf(
		"%s: user data retrieval is disabled for this provider.\n"+
			"Falling back to  %s provider",
		p.String(),
		p.providerOVF.String())
	return p.providerOVF.Extract()

}
