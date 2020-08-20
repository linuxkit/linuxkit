package main

import (
	"encoding/base64"
	"fmt"
	"path"

	"github.com/go-resty/resty"
)

// OracleInstanceMetaData oracle instance metadata
type OracleInstanceMetaData struct {
	AvailabilityDomain string  `json:"availabilityDomain"`
	FaultDomain        *string `json:"faultDomain,omitempty"`
	DisplayName        *string `json:"displayName,omitempty"`
	Hostname           string  `json:"hostname"`
	ID                 string  `json:"id"`
	Image              string  `json:"image"`
	Metadata           *struct {
		SSHAuthorizedKeys *string `json:"ssh_authorized_keys,omitempty"`
		UserData          *string `json:"user_data,omitempty"` // aka cloud-init data but in base64 form
	} `json:"metadata,omitempty"`
	Region              string  `json:"region"`
	CanonicalRegionName *string `json:"canonicalRegionName,omitempty"`
	Shape               string  `json:"shape"`
}

const (
	oracleMetadataBaseURL     = "http://169.254.169.254/opc/v1"
	oracleInstanceMetaDataURL = oracleMetadataBaseURL + "/instance/"
	// oracleNetworkMetaDataURL  = oracleMetadataBaseURL + "/vnics/" // TODO: Find a way to figure out public IP address
	// You can attach multiple NICs in OCI so...Also if you use ephemeral IP there is no public IP shown in this API call too
)

// An example metadata can be found here: https://docs.cloud.oracle.com/en-us/iaas/Content/Compute/Tasks/gettingmetadata.htm

// ProviderOracle is the type implementing the Provider interface for Oracle
type ProviderOracle struct {
	DefaultProviderUtil
	client *resty.Client
}

// NewOracle returns a new ProviderOracle
func NewOracle() *ProviderOracle {
	return &ProviderOracle{
		client: resty.New(),
	}
}

func (p *ProviderOracle) String() string {
	return "Oracle Cloud Infrastructure"
}

// ShortName short name
func (p *ProviderOracle) ShortName() string {
	return "oracle"
}

// Probe checks if we are running on Oracle
func (p *ProviderOracle) Probe() bool {
	// Getting the index should always work...
	res, err := p.client.R().Get(oracleInstanceMetaDataURL)
	return err == nil && res.IsSuccess()
}

// Extract gets both the Oracle specific and generic userdata
func (p *ProviderOracle) Extract() ([]byte, error) {
	resp, err := p.client.R().Get("http://checkip.amazonaws.com")
	if err == nil {
		if err := p.WriteDataToFile("public ipv4", 0644, string(resp.Body()), path.Join(ConfigPath, "public_ipv4")); err != nil {
			return nil, err
		}
	} else {
		fmt.Printf("oracle: error on getting a public ip address, this instance is probably private-only")
	}

	resp, err = p.client.R().SetResult(&OracleInstanceMetaData{}).Get(oracleInstanceMetaDataURL)
	if err != nil {
		return nil, err
	}

	if metadata, ok := resp.Result().(*OracleInstanceMetaData); ok && metadata != nil {
		if err := p.WriteDataToFile("instance id", 0644, metadata.ID, path.Join(ConfigPath, "instance_id")); err != nil {
			return nil, err
		}

		// there's three major shapes in oracle cloud, flexible, bare metal and VM, so it is definitely the instance type
		if err := p.WriteDataToFile("instance type", 0644, metadata.Shape, path.Join(ConfigPath, "instance_type")); err != nil {
			return nil, err
		}

		if err := p.WriteDataToFile("region", 0644, metadata.Region, path.Join(ConfigPath, "region")); err != nil {
			return nil, err
		}

		if err := p.WriteDataToFile("instance image", 0644, metadata.Image, path.Join(ConfigPath, "image")); err != nil {
			return nil, err
		}

		if err := p.WriteDataToFile("host name", 0644, metadata.Hostname, path.Join(ConfigPath, Hostname)); err != nil {
			return nil, err
		}

		if err := p.WriteDataToFile("availability domain", 0644, metadata.AvailabilityDomain, path.Join(ConfigPath, "availability_domain")); err != nil {
			return nil, err
		}

		if metadata.DisplayName != nil {
			if err := p.WriteDataToFile("local host name", 0644, *metadata.DisplayName, path.Join(ConfigPath, "local_hostname")); err != nil {
				return nil, err
			}
		}

		// well, sh*t, oracle sub-partition availability domain into fault domain,
		// so there's no concept of "zone" I don't know which is which,
		// in the meanwhile I think the canonical name would be the closest thing to availability zone, so whatever
		if metadata.CanonicalRegionName != nil {
			if err := p.WriteDataToFile(`"availability zone"`, 0644, *metadata.CanonicalRegionName, path.Join(ConfigPath, "availability_zone")); err != nil {
				return nil, err
			}
		}

		if metadata.FaultDomain != nil {
			if err := p.WriteDataToFile("fault domain", 0644, *metadata.FaultDomain, path.Join(ConfigPath, "fault_domain")); err != nil {
				return nil, err
			}
		}

		if userMetadata := metadata.Metadata; userMetadata != nil {
			// interestingly oracle set this as a metadata and will be undefined if the ssh key is not specified
			if userMetadata.SSHAuthorizedKeys != nil {
				if err := p.MakeFolder("ssh public keys", 0755, path.Join(ConfigPath, SSH)); err != nil {
					return nil, err
				}

				if err := p.WriteDataToFile("ssh public keys", 0600, *userMetadata.SSHAuthorizedKeys, path.Join(ConfigPath, SSH, "authorized_keys")); err != nil {
					return nil, err
				}
			}
			if userMetadata.UserData != nil {
				return base64.StdEncoding.DecodeString(*userMetadata.UserData)
			}
		}
	}
	return nil, nil
}
