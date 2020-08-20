package main

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/sl1pm4t/snooze"
)

// IkoulaMetadataAPI Ikoula metadata API
type IkoulaMetadataAPI struct {
	AvailabilityZone func() (string, error) `method:"GET" path:"/meta-data/availability-zone"`
	CloudIdentifier  func() (string, error) `method:"GET" path:"/meta-data/cloud-identifier"`
	Hostname         func() (string, error) `method:"GET" path:"/meta-data/hostname"`
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
	// and...this is why I said ikoula, or actually Apache CloudStack is so stupid
	// using DHCP server as metadata server? Oh homie please.
	possibleDhcpServAddr, err := GuessDHCPServerAddress()
	finalDhcpServer := "data-server."
	if err != nil {
		logrus.Errorf("%s: cannot find DHCP server from good known lease file locations: %v", provider.ShortName(), err)
	} else if len(possibleDhcpServAddr) > 1 {
		logrus.Warnf("%s: more than one DHCP server found. this is very unusual. speculatively using the first server found", provider.ShortName())
		finalDhcpServer = possibleDhcpServAddr[0]
	}

	client := snooze.Client{Root: finalDhcpServer}
	api := &IkoulaMetadataAPI{}
	client.Create(api)
	return &ProviderIkoula{IkoulaMetadataAPI: api}
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
	res, err := p.Hostname()
	return err == nil && res != ""
}

// Extract extract
func (p *ProviderIkoula) Extract() (userData []byte, err error) {
	var ret string

	if ret, err = p.InstanceID(); err == nil {
		if err := p.WriteDataToFile("instance id", 0644, ret, path.Join(ConfigPath, "instance_id")); err != nil {
			return nil, err
		}
	} else {
		logrus.Error("failed to get instance id: %s", err)
	}

	if ret, err = p.ServiceOffering(); err == nil {
		if err := p.WriteDataToFile("instance type", 0644, ret, path.Join(ConfigPath, "instance_type")); err != nil {
			return nil, err
		}
	} else {
		logrus.Error("failed to get instance type: %s", err)
	}

	if ret, err = p.AvailabilityZone(); err == nil {
		if err := p.WriteDataToFile("availability zone", 0644, ret, path.Join(ConfigPath, "availability_zone")); err != nil {
			return nil, err
		}
	} else {
		logrus.Error("failed to get availability zone: %s", err)
	}

	if ret, err = p.Hostname(); err == nil {
		if err := p.WriteDataToFile("host name", 0644, ret, path.Join(ConfigPath, Hostname)); err != nil {
			return nil, err
		}
	} else {
		logrus.Error("failed to get host name: %s", err)
	}

	if ret, err = p.LocalHostName(); err == nil {
		if err := p.WriteDataToFile("local host name", 0644, ret, path.Join(ConfigPath, "local-hostname")); err != nil {
			return nil, err
		}
	} else {
		logrus.Error("failed to get local host name: %s", err)
	}

	if ret, err = p.PublicIP(); err == nil {
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

const (
	// DHCPDirsPattern This is to match dhcp and dhcp3 and various many others
	DHCPDirsPattern = "/var/lib/dhcp*/*.leases"
)

// DHCPServerRegex I do not care about whether the IP address is correct, rather I assumed the DHCP client
// will always produce and validate the DHCP server to be in octet range
var DHCPServerRegex = regexp.MustCompile(`option[ \t]+dhcp-server-identifier[ \t]+(([0-9]{1,3}\.){3}[0-9]{1,3});?`)

// GuessDHCPServerAddress Guesses possible DHCP server addresses
// Used here to figure out the metadata service server
func GuessDHCPServerAddress() (possibleAddresses []string, err error) {
	// i actually wanted set data structure in golang...and golang team said barely anybody use it so nope
	// f*ck it, i roll my own. ken thompson why would you make such a stupid mistake?
	possibleAddressesHash := make(map[string]bool)
	defer func() {
		if err == nil {
			for key := range possibleAddressesHash {
				possibleAddresses = append(possibleAddresses, key)
			}
		}
	}()

	var matches []string
	// error or no match
	if matches, err = filepath.Glob(DHCPDirsPattern); err != nil || len(matches) < 1 {
		return
	}

	for _, value := range matches {
		// fail fast if there's an error already
		if err != nil {
			return
		}
		func() {
			var (
				file *os.File
				stat os.FileInfo
				data []byte
			)
			// can't really use := because name shadowing
			if file, err = os.Open(value); err != nil {
				return
			}
			if stat, err = file.Stat(); err != nil {
				return
			}
			// since we work on linux exclusive platform this is fine
			if data, err = syscall.Mmap(int(file.Fd()), 0, int(stat.Size()), syscall.PROT_READ, syscall.MAP_SHARED); err != nil {
				return
			}
			defer func() {
				if err = syscall.Munmap(data); err != nil {
					return
				}
			}()

			// submatch 0 is the matched expression source
			// so starting from submatch 1 it will be the capture groups...at least 2 elements
			if submatches := DHCPServerRegex.FindSubmatch(data); len(submatches) > 1 {
				for _, submatch := range submatches[1:] {
					ipAddr := string(submatch)
					// cheap check without having to insert multiple times...
					if _, ok := possibleAddressesHash[ipAddr]; !ok {
						possibleAddressesHash[ipAddr] = true
					}
				}
			}
		}()
	}
	return
}
