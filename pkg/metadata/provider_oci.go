package main

import (
	"encoding/base64"
	"path"

	"github.com/go-resty/resty"
	"github.com/sirupsen/logrus"
	// "gopkg.in/resty.v1"
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
func NewOracle() (provider *ProviderOracle) {
	defer func() { provider.ProviderUtil = provider }()
	return &ProviderOracle{client: resty.New()}
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
func (p *ProviderOracle) Extract() (userData []byte, err error) {
	var publicIP string
	resp, err := p.client.R().Get("http://checkip.amazonaws.com")
	if err == nil {
		publicIP = string(resp.Body())
	} else {
		logrus.WithError(err).
			Error("cannot get public ip address. this instance is probably NAT-only")
	}

	resp, err = p.client.R().SetResult(&OracleInstanceMetaData{}).Get(oracleInstanceMetaDataURL)
	if err != nil {
		return
	}

	if metadata, ok := resp.Result().(*OracleInstanceMetaData); ok && metadata != nil {
		// wrap a variable/field to match simpleExtractData.Getter
		wrap := func(x *string) func() (string, error) {
			return func() (ret string, err error) {
				if x != nil {
					ret = *x
				} else {
					ret = ""
				}
				return
			}
		}
		err = p.simpleExtract([]simpleExtractData{
			{Type: "region", Dest: path.Join(ConfigPath, "region"), Perm: 0644, Getter: wrap(&metadata.Region)},
			// well, sh*t, oracle sub-partition availability domain into fault domain,
			// so there's no concept of "zone" I don't know which is which,
			// in the meanwhile I think the canonical name would be the closest thing to availability zone, so whatever
			{Type: "availability zone", Dest: path.Join(ConfigPath, "availability_zone"), Perm: 0644, Getter: wrap(metadata.CanonicalRegionName)},
			{Type: "availability domain", Dest: path.Join(ConfigPath, "domain", "availability"), Perm: 0644, Getter: wrap(&metadata.AvailabilityDomain)},
			{Type: "fault domain", Dest: path.Join(ConfigPath, "domain", "fault"), Perm: 0644, Getter: wrap(metadata.FaultDomain)},
			{Type: "instance type", Dest: path.Join(ConfigPath, "instance_type"), Perm: 0644, Getter: wrap(&metadata.Shape)},
			{Type: "instance image", Dest: path.Join(ConfigPath, "image"), Perm: 0644, Getter: wrap(&metadata.Image)},
			{Type: "instance id", Dest: path.Join(ConfigPath, "instance_id"), Perm: 0644, Getter: wrap(&metadata.ID)},
			{Type: "host name", Dest: path.Join(ConfigPath, Hostname), Perm: 0644, Getter: wrap(&metadata.Hostname)},
			{Type: "local host name", Dest: path.Join(ConfigPath, "local_host_name"), Perm: 0644, Getter: wrap(metadata.DisplayName)},
			{Type: "public ipv4", Dest: path.Join(ConfigPath, "public_ipv4"), Perm: 0644, Getter: wrap(&publicIP)},
			{Type: "ssh public keys", Dest: path.Join(ConfigPath, SSH, "authorized_keys"), Perm: 0755, Success: ensureSSHKeySecure,
				Getter: func() (ret string, err error) {
					if userMetadata := metadata.Metadata; userMetadata != nil {
						// interestingly oracle set this as a metadata and will be undefined if the ssh key is not specified
						if userMetadata.SSHAuthorizedKeys != nil {
							ret = *userMetadata.SSHAuthorizedKeys
						}
					}
					return
				},
			},
		})
		if err == nil {
			if userMetadata := metadata.Metadata; userMetadata != nil {
				if optUserData := userMetadata.UserData; optUserData != nil {
					userData, err = base64.StdEncoding.DecodeString(*optUserData)
				}
			}
		}
	}
	return
}
