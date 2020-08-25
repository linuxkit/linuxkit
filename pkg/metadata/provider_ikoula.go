package main

import (
	"fmt"
	"path"

	"github.com/sl1pm4t/snooze"
)

// IkoulaMetadataAPI Ikoula metadata API
type IkoulaMetadataAPI struct {
	AvailabilityZone func() (string, error) `method:"GET" path:"/meta-data/availability-zone"`
	CloudIdentifier  func() (string, error) `method:"GET" path:"/meta-data/cloud-identifier"`
	InstanceID       func() (string, error) `method:"GET" path:"/meta-data/instance-id"`
	LocalHostName    func() (string, error) `method:"GET" path:"/meta-data/local-hostname"`
	PrivateIP        func() (string, error) `method:"GET" path:"/meta-data/local-ipv4"`
	PublicHostName   func() (string, error) `method:"GET" path:"/meta-data/public-hostname"`
	PublicIP         func() (string, error) `method:"GET" path:"/meta-data/public-ipv4"`
	PublicKeys       func() (string, error) `method:"GET" path:"/meta-data/public-keys"`
	ServiceOffering  func() (string, error) `method:"GET" path:"/meta-data/service-offering"`
	Userdata         func() (string, error) `method:"GET" path:"/user-data"`
	VMIdentifier     func() (string, error) `method:"GET" path:"/meta-data/vm-id"`
}

// ProviderIkoula Ikoula provider
// a public/private/hybrid cloud provider from France building upon the arse of Apache CloudStack. Bonjour!
type ProviderIkoula struct {
	DefaultProviderUtil
	*IkoulaMetadataAPI
}

// NewIkoula new ikoula provider
func NewIkoula() (provider *ProviderIkoula) {
	provider = &ProviderIkoula{}
	provider.ProviderUtil = provider
	// and...this is why I said ikoula, or actually Apache CloudStack is so stupid
	// using DHCP server as metadata server? Oh homie please.
	finalDhcpServer := "data-server."
	defer func() {
		client := snooze.Client{Root: fmt.Sprintf("http://%s/latest", finalDhcpServer)}
		snoozeSetDefaultLogger(&client)
		api := &IkoulaMetadataAPI{}
		client.Create(api)
		provider.IkoulaMetadataAPI = api
	}()

	possibleDhcpServAddr, err := FindPossibleDHCPServers()
	if err == nil && len(possibleDhcpServAddr) > 0 {
		if len(possibleDhcpServAddr) > 1 {
			provider.PrepareLogger().
				Warn("more than one DHCP server found. this is very unusual")
		}
		finalDhcpServer = possibleDhcpServAddr[0]
	} else {
		provider.PrepareLogger().
			Error("failed to guess DHCP server. using the fallback DNS hostname as default")
	}
	return
}

func (p *ProviderIkoula) String() string {
	return "Ikoula Cloud"
}

// ShortName short name
func (p *ProviderIkoula) ShortName() string {
	return "ikoula"
}

// Probe probe
func (p *ProviderIkoula) Probe() bool {
	res, err := p.CloudIdentifier()
	return err == nil && res != ""
}

// Extract extract
func (p *ProviderIkoula) Extract() (userData []byte, err error) {
	err = p.simpleExtract([]simpleExtractData{
		{Type: "instance id", Dest: path.Join(ConfigPath, "instance_id"), Perm: 0644, Getter: p.InstanceID},
		{Type: "instance type", Dest: path.Join(ConfigPath, "instance_type"), Perm: 0644, Getter: p.ServiceOffering},
		{Type: "availability zone", Dest: path.Join(ConfigPath, "availability_zone"), Perm: 0644, Getter: p.AvailabilityZone},
		{Type: "host name", Dest: path.Join(ConfigPath, Hostname), Perm: 0644, Getter: p.LocalHostName},
		{Type: "public host name", Dest: path.Join(ConfigPath, "public_host_name"), Perm: 0644, Getter: p.PublicHostName},
		{Type: "public ipv4", Dest: path.Join(ConfigPath, "public_ipv4"), Perm: 0644, Getter: p.PublicIP},
		{Type: "private ipv4", Dest: path.Join(ConfigPath, "private_ipv4"), Perm: 0644, Getter: p.PrivateIP},
		{Type: "ssh public keys", Dest: path.Join(ConfigPath, SSH, "authorized_keys"), Perm: 0755, Getter: p.PublicKeys, Success: ensureSSHKeySecure},
	})
	if err == nil {
		var ret string
		ret, err = p.Userdata()
		userData = []byte(ret)
	}
	return
}
