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
func NewAzure() (provider *ProviderAzure) {
	defer func() { provider.ProviderUtil = provider }()
	client := snooze.Client{Root: azureAPIBaseURL, Before: func(request *retryablehttp.Request, client *retryablehttp.Client) {
		request.Header.Add("Metadata", "true")
		q := request.URL.Query()
		q.Add("api-version", azureAPIVersion)
		request.URL.RawQuery = q.Encode()
	}}
	snoozeSetDefaultLogger(&client)
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
func (p *ProviderAzure) Extract() (userData []byte, err error) {
	// wrap a variable/field to match simpleExtractData.Getter
	wrap := func(x *string) func() (string, error) {
		return func() (ret string, err error) {
			// screw you, golang, where's my ternary expression?
			if x != nil {
				ret = *x
			} else {
				ret = ""
			}
			return
		}
	}
	var (
		networkMetadata AzureInstanceNetworkMetadata
		computeMetadata AzureInstanceComputeMetadata
	)

	networkMetadata, err = p.NetworkMetadata()
	if err != nil {
		return
	}
	computeMetadata, err = p.ComputeMetadata()
	if err != nil {
		return
	}
	ip := networkMetadata.Interface[0].Ipv4.IPAddress[0]

	err = p.simpleExtract([]simpleExtractData{
		{Type: "region", Dest: path.Join(ConfigPath, "region"), Perm: 0644, Getter: wrap(&computeMetadata.Location)},
		{Type: "availability zone", Dest: path.Join(ConfigPath, "availability_zone"), Perm: 0644, Getter: wrap(&computeMetadata.Zone)},
		{Type: "instance type", Dest: path.Join(ConfigPath, "instance_type"), Perm: 0644, Getter: wrap(&computeMetadata.VMSize)},
		{Type: "instance image", Dest: path.Join(ConfigPath, "image"), Perm: 0644, Getter: wrap(&computeMetadata.StorageProfile.OsDisk.Name)},
		{Type: "instance id", Dest: path.Join(ConfigPath, "instance_id"), Perm: 0644, Getter: wrap(&computeMetadata.VMID)},
		// unfortunately azure assumes the vm name to be the instance host name
		{Type: "host name", Dest: path.Join(ConfigPath, Hostname), Perm: 0644, Getter: wrap(&computeMetadata.Name)},
		{Type: "private ipv4", Dest: path.Join(ConfigPath, "private_ipv4"), Perm: 0644, Getter: wrap(&ip.PrivateIPAddress)},
		{Type: "public ipv4", Dest: path.Join(ConfigPath, "public_ipv4"), Perm: 0644, Getter: wrap(ip.PublicIPAddress)},
		{Type: "ssh public keys", Dest: path.Join(ConfigPath, SSH, "authorized_keys"), Perm: 0755, Success: ensureSSHKeySecure,
			Getter: func() (ret string, err error) {
				if publicKeys := computeMetadata.PublicKeys; publicKeys != nil && len(*publicKeys) > 0 {
					ret = strings.Join(funk.Map(publicKeys, func(entry AzurePublicKeyEntry) string { return entry.KeyData }).([]string), "\n")
				}
				return
			},
		},
	})
	if err == nil {
		// TODO: this field is disabled, figure out a way to obtain user data somewhere else
		userData = []byte(computeMetadata.CustomData)
	}
	return
}
