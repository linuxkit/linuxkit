package main

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"path/filepath"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

// ProviderAzureOVF extracts user data from ovf-env.xml. The file is on an Azure
// attached DVD containing the user data, encoded as base64.
// Inspired by:
//  - WALinuxAgent:
//	- https://github.com/Azure/WALinuxAgent
//  - cloud-init Azure Datasource docs:
//    https://cloudinit.readthedocs.io/en/latest/topics/datasources/azure.html
type ProviderAzureOVF struct {
	client     *http.Client
	mountPoint string
}

// OVF XML model
type OVF struct {
	HostName       string `xml:"ProvisioningSection>LinuxProvisioningConfigurationSet>HostName"`
	UserDataBase64 string `xml:"ProvisioningSection>LinuxProvisioningConfigurationSet>CustomData"`
}

// NewAzureOVF factory
func NewAzureOVF(client *http.Client) *ProviderAzureOVF {
	mountPoint, err := ioutil.TempDir("", "cdrom")
	if err != nil {
		panic(fmt.Sprintf("Azure-OVF: Creating temp mount dir failed: %s", err))
	}
	return &ProviderAzureOVF{
		mountPoint: mountPoint,
		client:     client,
	}
}

func (p *ProviderAzureOVF) String() string {
	return "Azure-OVF"
}

// Probe returns true if DVD is successfully mounted, false otherwise
func (p *ProviderAzureOVF) Probe() bool {
	if err := retry(6, 5*time.Second, p.mount); err != nil {
		log.Debugf("%s: Probe failed: %s", p.String(), err)
		return false
	}
	return true
}

// Extract user data from ovf-env.xml file located on Azure attached DVD
func (p *ProviderAzureOVF) Extract() ([]byte, error) {
	ovf, err := p.getOVF()
	if err != nil {
		return nil, fmt.Errorf("%s: %s", p.String(), err)
	}

	if err = p.saveHostname(ovf); err != nil {
		return nil, fmt.Errorf("%s: %s", p.String(), err)
	}

	// TODO get SSH keys

	userData, err := p.getUserData(ovf)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", p.String(), err)
	}
	return userData, nil
}

// Mount DVD attached by Azure
func (p *ProviderAzureOVF) mount() error {
	dev, err := getDvdDevice()
	if err != nil {
		return err
	}
	// Read-only mount UDF file system
	// https://github.com/Azure/WALinuxAgent/blob/v2.2.52/azurelinuxagent/common/osutil/default.py#L602-L605
	return syscall.Mount(dev, p.mountPoint, "udf", syscall.MS_RDONLY, "")
}

// WALinuxAgent implements various methods of finding the DVD device, depending
// on the OS:
// https://github.com/Azure/WALinuxAgent/blob/develop/azurelinuxagent/common/osutil
func getDvdDevice() (string, error) {
	var (
		// Default WALinuxAgent implementation, see:
		// https://github.com/Azure/WALinuxAgent/blob/v2.2.52/azurelinuxagent/common/osutil/default.py#L569
		dvdPatterns = []string{
			"/dev/sr[0-9]",
			"/dev/hd[c-z]",
			"/dev/cdrom[0-9]",
		}
	)
	for _, pattern := range dvdPatterns {
		devs, err := filepath.Glob(pattern)
		if err != nil {
			panic(fmt.Sprintf("Invalid glob pattern: %s", pattern))
		}
		if len(devs) > 0 {
			log.Debugf("Found DVD device: %s", devs[0])
			return devs[0], nil
		}
	}
	return "", fmt.Errorf("No DVD device found")
}

func (p *ProviderAzureOVF) getOVF() (*OVF, error) {
	const (
		fileName = "ovf-env.xml"
	)
	xmlContent, err := ioutil.ReadFile(path.Join(p.mountPoint, fileName))
	if err != nil {
		return nil, fmt.Errorf("Reading file %s failed: %s", fileName, err)
	}
	err = ioutil.WriteFile(path.Join(ConfigPath, fileName), xmlContent, 0600)
	if err != nil {
		return nil, fmt.Errorf("Copying file %s failed: %s", fileName, err)
	}

	defer p.unmount()

	var ovf OVF
	err = xml.Unmarshal(xmlContent, &ovf)
	if err != nil {
		return nil, fmt.Errorf("Unmarshalling %s failed: %s", fileName, err)
	}
	return &ovf, nil
}

// Unmount DVD attached by Azure
func (p *ProviderAzureOVF) unmount() {
	_ = syscall.Unmount(p.mountPoint, 0)
}

func (p *ProviderAzureOVF) saveHostname(ovf *OVF) error {
	if ovf == nil || ovf.HostName == "" {
		return fmt.Errorf("Hostname is empty")
	}
	err := ioutil.WriteFile(path.Join(ConfigPath, Hostname), []byte(ovf.HostName), 0644)
	if err != nil {
		return fmt.Errorf("Failed to write hostname: %s", err)
	}
	log.Debugf("%s: Saved hostname: %s", p.String(), ovf.HostName)
	return nil
}

func (p *ProviderAzureOVF) getUserData(ovf *OVF) ([]byte, error) {
	defer ReportReady(p.client)

	if ovf == nil || ovf.UserDataBase64 == "" {
		log.Debugf("%s: User data is empty", p.String())
		return nil, nil
	}
	log.Debugf("%s: Base64 user data: %s", p.String(), ovf.UserDataBase64)
	userData, err := base64.StdEncoding.DecodeString(ovf.UserDataBase64)
	if err != nil {
		return nil, fmt.Errorf("Decoding user data failed: %s", err)
	}
	log.Debugf("%s: Raw user data: \n%s", p.String(), string(userData))
	return userData, nil
}

// https://stackoverflow.com/questions/47606761/repeat-code-if-an-error-occured/47606858#47606858
// https://blog.abourget.net/en/2016/01/04/my-favorite-golang-retry-function/
func retry(attempts int, sleep time.Duration, f func() error) (err error) {
	for i := 0; ; i++ {
		err = f()
		if err == nil {
			return
		}

		if i >= (attempts - 1) {
			break
		}

		time.Sleep(sleep)

		log.Debugf("Retrying after error: %s", err)
	}
	return fmt.Errorf("After %d attempts, last error: %s", attempts, err)
}
