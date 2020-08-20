package main

import (
	"path"

	"github.com/sirupsen/logrus"
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
	client := snooze.Client{Root: aliCloudMetadataBaseURL}
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
	res, err := p.Hostname()
	return err == nil && res != ""
}

// Extract extract
func (p *ProviderAliCloud) Extract() (userData []byte, err error) {
	var ret string

	if ret, err = p.InstanceID(); err == nil {
		if err := p.WriteDataToFile("instance id", 0644, ret, path.Join(ConfigPath, "instance_id")); err != nil {
			return nil, err
		}
	} else {
		logrus.Error("failed to get instance id: %s", err)
	}

	if ret, err = p.InstanceType(); err == nil {
		if err := p.WriteDataToFile("instance type", 0644, ret, path.Join(ConfigPath, "instance_type")); err != nil {
			return nil, err
		}
	} else {
		logrus.Error("failed to get instance type: %s", err)
	}

	if ret, err = p.Region(); err == nil {
		if err := p.WriteDataToFile("region", 0644, ret, path.Join(ConfigPath, "region")); err != nil {
			return nil, err
		}
	} else {
		logrus.Error("failed to get region: %s", err)
	}

	if ret, err = p.Zone(); err == nil {
		if err := p.WriteDataToFile("zone", 0644, ret, path.Join(ConfigPath, "zone")); err != nil {
			return nil, err
		}
	} else {
		logrus.Error("failed to get zone: %s", err)
	}

	if ret, err = p.ImageID(); err == nil {
		if err := p.WriteDataToFile("instance image", 0644, ret, path.Join(ConfigPath, "image")); err != nil {
			return nil, err
		}
	} else {
		logrus.Error("failed to get instance image: %s", err)
	}

	if ret, err = p.Hostname(); err == nil {
		if err := p.WriteDataToFile("host name", 0644, ret, path.Join(ConfigPath, Hostname)); err != nil {
			return nil, err
		}
	} else {
		logrus.Error("failed to get host name: %s", err)
	}

	if ret, err = p.PublicIP(); err != nil {
		ret, err = p.ElasticPublicIP()
	}

	if err == nil {
		if err := p.WriteDataToFile("public ipv4", 0644, ret, path.Join(ConfigPath, "public_ipv4")); err != nil {
			return nil, err
		}
	} else {
		logrus.Error("failed to get public ipv4: %s", err)
	}

	if ret, err = p.PrivateIP(); err == nil {
		if err := p.WriteDataToFile("private ipv4", 0644, ret, path.Join(ConfigPath, "private_ipv4")); err != nil {
			return nil, err
		}
	} else {
		logrus.Error("failed to get private ipv4: %s", err)
	}

	if ret, err = p.PublicKeys(); err == nil {
		if err = p.MakeFolder("ssh public keys", 0755, path.Join(ConfigPath, SSH)); err != nil {
			return nil, err
		}

		if err = p.WriteDataToFile("ssh public keys", 0600, ret, path.Join(ConfigPath, SSH, "authorized_keys")); err != nil {
			return nil, err
		}
	} else {
		logrus.Error("failed to get public keys: %s", err)
	}

	ret, err = p.Userdata()
	return []byte(ret), err
}
