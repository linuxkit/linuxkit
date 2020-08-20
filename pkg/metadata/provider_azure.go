package main

import (
	"path"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/sl1pm4t/snooze"
	"github.com/thoas/go-funk"
)

const (
	azureAPIBaseURL = "http://169.254.169.254/metadata/instance"
	azureAPIVersion = "2020-06-01"
)

// AzurePublicKeyEntry public key entry
type AzurePublicKeyEntry struct {
	KeyData string `json:"keyData"`
	Path    string `json:"path"`
}

// AzureInstanceComputeMetadata compute metadata
type AzureInstanceComputeMetadata struct {
	AzEnvironment              string `json:"azEnvironment"`
	CustomData                 string `json:"customData"`
	IsHostCompatibilityLayerVM string `json:"isHostCompatibilityLayerVm"`
	Location                   string `json:"location"`
	Name                       string `json:"name"`
	Offer                      string `json:"offer"`
	OsType                     string `json:"osType"`
	PlacementGroupID           string `json:"placementGroupId"`
	Plan                       struct {
		Name      string `json:"name"`
		Product   string `json:"product"`
		Publisher string `json:"publisher"`
	} `json:"plan"`
	PlatformFaultDomain  string                 `json:"platformFaultDomain"`
	PlatformUpdateDomain string                 `json:"platformUpdateDomain"`
	Provider             string                 `json:"provider"`
	PublicKeys           *[]AzurePublicKeyEntry `json:"publicKeys,omitempty"`
	Publisher            string                 `json:"publisher"`
	ResourceGroupName    string                 `json:"resourceGroupName"`
	ResourceID           string                 `json:"resourceId"`
	SecurityProfile      struct {
		SecureBootEnabled string `json:"secureBootEnabled"`
		VirtualTpmEnabled string `json:"virtualTpmEnabled"`
	} `json:"securityProfile"`
	Sku            string `json:"sku"`
	StorageProfile struct {
		DataDisks      []interface{} `json:"dataDisks"`
		ImageReference struct {
			ID        string `json:"id"`
			Offer     string `json:"offer"`
			Publisher string `json:"publisher"`
			Sku       string `json:"sku"`
			Version   string `json:"version"`
		} `json:"imageReference"`
		OsDisk struct {
			Caching          string `json:"caching"`
			CreateOption     string `json:"createOption"`
			DiffDiskSettings struct {
				Option string `json:"option"`
			} `json:"diffDiskSettings"`
			DiskSizeGB         string `json:"diskSizeGB"`
			EncryptionSettings struct {
				Enabled string `json:"enabled"`
			} `json:"encryptionSettings"`
			Image struct {
				URI string `json:"uri"`
			} `json:"image"`
			ManagedDisk struct {
				ID                 string `json:"id"`
				StorageAccountType string `json:"storageAccountType"`
			} `json:"managedDisk"`
			Name   string `json:"name"`
			OsType string `json:"osType"`
			Vhd    struct {
				URI string `json:"uri"`
			} `json:"vhd"`
			WriteAcceleratorEnabled string `json:"writeAcceleratorEnabled"`
		} `json:"osDisk"`
	} `json:"storageProfile"`
	SubscriptionID string        `json:"subscriptionId"`
	Tags           string        `json:"tags"`
	TagsList       []interface{} `json:"tagsList"`
	Version        string        `json:"version"`
	VMID           string        `json:"vmId"`
	VMScaleSetName string        `json:"vmScaleSetName"`
	VMSize         string        `json:"vmSize"`
	Zone           string        `json:"zone"`
}

// AzureInstanceNetworkMetadata ...
type AzureInstanceNetworkMetadata struct {
	Interface []struct {
		Ipv4 struct {
			IPAddress []struct {
				PrivateIPAddress string  `json:"privateIpAddress"`
				PublicIPAddress  *string `json:"publicIpAddress,omitempty"`
			} `json:"ipAddress"`
			Subnet []struct {
				Address string `json:"address"`
				Prefix  string `json:"prefix"`
			} `json:"subnet"`
		} `json:"ipv4"`
		Ipv6 struct {
			IPAddress []interface{} `json:"ipAddress"`
		} `json:"ipv6"`
		MacAddress string `json:"macAddress"`
	} `json:"interface"`
}

// AzureMetadataAPI metadata API
type AzureMetadataAPI struct {
	ComputeMetadata func() (AzureInstanceComputeMetadata, error) `method:"GET" path:"/compute"`
	NetworkMetadata func() (AzureInstanceNetworkMetadata, error) `method:"GET" path:"/network"`
}

// ProviderAzure azure provider
type ProviderAzure struct {
	DefaultProviderUtil
	*AzureMetadataAPI
}

// NewAzure new azure provider
func NewAzure() *ProviderAzure {
	client := snooze.Client{Root: azureAPIBaseURL, Before: func(request *retryablehttp.Request, client *retryablehttp.Client) {
		request.Header.Add("Metadata", "true")
		q := request.URL.Query()
		q.Add("api-version", azureAPIVersion)
		request.URL.RawQuery = q.Encode()
	}}
	api := &AzureMetadataAPI{}
	client.Create(api)
	return &ProviderAzure{AzureMetadataAPI: api}
}

func (p *ProviderAzure) String() string {
	return "Azure"
}

// ShortName short name
func (p *ProviderAzure) ShortName() string {
	return "azure"
}

// Probe probe
func (p *ProviderAzure) Probe() bool {
	_, err := p.ComputeMetadata()
	return err == nil
}

// Extract extract
func (p *ProviderAzure) Extract() ([]byte, error) {

	if metadata, err := p.NetworkMetadata(); err == nil {
		ip := metadata.Interface[0].Ipv4.IPAddress[0]

		if ip.PublicIPAddress != nil {
			if err := p.WriteDataToFile("public ipv4", 0644, *ip.PublicIPAddress, path.Join(ConfigPath, "public_ipv4")); err != nil {
				return nil, err
			}
		}

		if err := p.WriteDataToFile("private ipv4", 0644, ip.PrivateIPAddress, path.Join(ConfigPath, "private_ipv4")); err != nil {
			return nil, err
		}
	}

	if metadata, err := p.ComputeMetadata(); err == nil {
		if err := p.WriteDataToFile("instance id", 0644, metadata.VMID, path.Join(ConfigPath, "instance_id")); err != nil {
			return nil, err
		}

		// there's three major shapes in oracle cloud, flexible, bare metal and VM, so it is definitely the instance type
		if err := p.WriteDataToFile("instance type", 0644, metadata.VMSize, path.Join(ConfigPath, "instance_type")); err != nil {
			return nil, err
		}

		if err := p.WriteDataToFile("region", 0644, metadata.Location, path.Join(ConfigPath, "region")); err != nil {
			return nil, err
		}

		if err := p.WriteDataToFile("instance image", 0644, metadata.StorageProfile.OsDisk.Name, path.Join(ConfigPath, "image")); err != nil {
			return nil, err
		}

		// unfortunately azure assumes the vm name to be the instance host name
		if err := p.WriteDataToFile("host name", 0644, metadata.Name, path.Join(ConfigPath, Hostname)); err != nil {
			return nil, err
		}

		if err := p.WriteDataToFile("availability zone", 0644, metadata.Zone, path.Join(ConfigPath, "availability_zone")); err != nil {
			return nil, err
		}

		if publicKeys := metadata.PublicKeys; publicKeys != nil && len(*publicKeys) > 0 {
			if err := p.MakeFolder("ssh public keys", 0755, path.Join(ConfigPath, SSH)); err != nil {
				return nil, err
			}

			combinedSSHKeys := strings.Join(funk.Map(publicKeys, func(entry AzurePublicKeyEntry) string {
				return entry.KeyData
			}).([]string), "\n")

			if err := p.WriteDataToFile("ssh public keys", 0600, combinedSSHKeys, path.Join(ConfigPath, SSH, "authorized_keys")); err != nil {
				return nil, err
			}
		}

		// TODO: this field is disabled, figure out a way to obtain user data somewhere else
		return []byte(metadata.CustomData), nil
	}

	return nil, nil
}
