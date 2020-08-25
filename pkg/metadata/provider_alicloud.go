package main

import (
	"os"
	"path"

	"github.com/sl1pm4t/snooze"
)

const (
	aliCloudMetadataBaseURL = "http://100.100.100.200/latest/"
)

// AliCloudMetadataAPI alicloud metadata API
type AliCloudMetadataAPI struct {
	ElasticPublicIP    func() (string, error) `method:"GET" path:"/meta-data/eipv4"`
	Hostname           func() (string, error) `method:"GET" path:"/meta-data/hostname"`
	ImageID            func() (string, error) `method:"GET" path:"/meta-data/image-id"`
	InstanceID         func() (string, error) `method:"GET" path:"/meta-data/instance-id"`
	InstanceMaxEgress  func() (string, error) `method:"GET" path:"/meta-data/instance/max-netbw-egress"`
	InstanceMaxIngress func() (string, error) `method:"GET" path:"/meta-data/instance/max-netbw-ingerss"`
	InstanceType       func() (string, error) `method:"GET" path:"/meta-data/instance/instance-type"`
	PrivateIP          func() (string, error) `method:"GET" path:"/meta-data/private-ipv4"`
	PublicIP           func() (string, error) `method:"GET" path:"/meta-data/public-ipv4"`
	PublicKeys         func() (string, error) `method:"GET" path:"/meta-data/public-keys"`
	Region             func() (string, error) `method:"GET" path:"/meta-data/region-id"`
	Userdata           func() (string, error) `method:"GET" path:"/user-data"`
	Zone               func() (string, error) `method:"GET" path:"/meta-data/zone-id"`
}

// ProviderAliCloud alicloud provider
type ProviderAliCloud struct {
	DefaultProviderUtil
	*AliCloudMetadataAPI
}

// NewAliCloud new alicloud provider
func NewAliCloud() (provider *ProviderAliCloud) {
	defer func() { provider.ProviderUtil = provider }()
	client := snooze.Client{Root: aliCloudMetadataBaseURL}
	snoozeSetDefaultLogger(&client)
	api := &AliCloudMetadataAPI{}
	client.Create(api)
	return &ProviderAliCloud{AliCloudMetadataAPI: api}
}

func (p *ProviderAliCloud) String() string {
	return "Alibaba Cloud"
}

// ShortName short name
func (p *ProviderAliCloud) ShortName() string {
	return "alicloud"
}

// Probe probe
func (p *ProviderAliCloud) Probe() bool {
	res, err := p.InstanceMaxIngress()
	return err == nil && res != ""
}

// Extract extract
func (p *ProviderAliCloud) Extract() (userData []byte, err error) {
	err = p.simpleExtract([]simpleExtractData{
		{Type: "instance id", Dest: path.Join(ConfigPath, "instance_id"), Perm: 0644, Getter: p.InstanceID},
		{Type: "instance type", Dest: path.Join(ConfigPath, "instance_type"), Perm: 0644, Getter: p.InstanceType},
		{Type: "region", Dest: path.Join(ConfigPath, "region"), Perm: 0644, Getter: p.Region},
		{Type: "zone", Dest: path.Join(ConfigPath, "zone"), Perm: 0644, Getter: p.Zone},
		{Type: "instance image", Dest: path.Join(ConfigPath, "image"), Perm: 0644, Getter: p.ImageID},
		{Type: "host name", Dest: path.Join(ConfigPath, Hostname), Perm: 0644, Getter: p.Hostname},
		{Type: "public ipv4", Dest: path.Join(ConfigPath, "public_ipv4"), Perm: 0644, Getter: func() (ret string, err error) {
			if ret, err = p.PublicIP(); err != nil {
				ret, err = p.ElasticPublicIP()
			}
			return
		}},
		{Type: "private ipv4", Dest: path.Join(ConfigPath, "private_ipv4"), Perm: 0644, Getter: p.PrivateIP},
		{Type: "ssh public keys", Dest: path.Join(ConfigPath, SSH, "authorized_keys"), Perm: 0755, Getter: p.PublicKeys, Success: func() error {
			return os.Chmod(path.Join(ConfigPath, SSH, "authorized_keys"), 0600)
		}},
	})
	if err != nil {
		return
	}
	var ret string
	ret, err = p.Userdata()
	userData = []byte(ret)
	return
}
