package metadata

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
)

const BaseURL = "https://metadata.packet.net"

func GetMetadata() (*CurrentDevice, error) {
	res, err := http.Get(BaseURL + "/metadata")
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, err
	}

	var result struct {
		Error string `json:"error"`
		*CurrentDevice
	}
	if err := json.Unmarshal(b, &result); err != nil {
		if res.StatusCode >= 400 {
			return nil, errors.New(res.Status)
		}
		return nil, err
	}
	if result.Error != "" {
		return nil, errors.New(result.Error)
	}
	return result.CurrentDevice, nil
}

func GetUserData() ([]byte, error) {
	res, err := http.Get(BaseURL + "/userdata")
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	return b, err
}

type AddressFamily int

const (
	IPv4 = AddressFamily(4)
	IPv6 = AddressFamily(6)
)

type AddressInfo struct {
	ID          string        `json:"id"`
	Family      AddressFamily `json:"address_family"`
	Public      bool          `json:"public"`
	Management  bool          `json:"management"`
	Address     net.IP        `json:"address"`
	NetworkMask net.IP        `json:"netmask"`
	Gateway     net.IP        `json:"gateway"`
	NetworkBits int           `json:"cidr"`

	// These are available, but not really needed:
	//   Network     net.IP `json:"network"`
}

type BondingMode int

const (
	BondingBalanceRR    = BondingMode(0)
	BondingActiveBackup = BondingMode(1)
	BondingBalanceXOR   = BondingMode(2)
	BondingBroadcast    = BondingMode(3)
	BondingLACP         = BondingMode(4)
	BondingBalanceTLB   = BondingMode(5)
	BondingBalanceALB   = BondingMode(6)
)

var bondingModeStrings = map[BondingMode]string{
	BondingBalanceRR:    "balance-rr",
	BondingActiveBackup: "active-backup",
	BondingBalanceXOR:   "balance-xor",
	BondingBroadcast:    "broadcast",
	BondingLACP:         "802.3ad",
	BondingBalanceTLB:   "balance-tlb",
	BondingBalanceALB:   "balance-alb",
}

func (m BondingMode) String() string {
	if str, ok := bondingModeStrings[m]; ok {
		return str
	}
	return fmt.Sprintf("%d", m)
}

type CurrentDevice struct {
	ID       string          `json:"id"`
	Hostname string          `json:"hostname"`
	IQN      string          `json:"iqn"`
	Plan     string          `json:"plan"`
	Facility string          `json:"facility"`
	Tags     []string        `json:"tags"`
	SSHKeys  []string        `json:"ssh_keys"`
	OS       OperatingSystem `json:"operating_system"`
	Network  NetworkInfo     `json:"network"`
	Volumes  []VolumeInfo    `json:"volume"`

	// This is available, but is actually inaccurate, currently:
	//   APIBaseURL string          `json:"api_url"`
}

type InterfaceInfo struct {
	Name string `json:"name"`
	MAC  string `json:"mac"`
}

func (i *InterfaceInfo) ParseMAC() (net.HardwareAddr, error) {
	return net.ParseMAC(i.MAC)
}

type NetworkInfo struct {
	Interfaces []InterfaceInfo `json:"interfaces"`
	Addresses  []AddressInfo   `json:"addresses"`

	Bonding struct {
		Mode BondingMode `json:"mode"`
	} `json:"bonding"`
}

func (n *NetworkInfo) BondingMode() BondingMode {
	return n.Bonding.Mode
}

type OperatingSystem struct {
	Slug    string `json:"slug"`
	Distro  string `json:"distro"`
	Version string `json:"version"`
}

type VolumeInfo struct {
	Name string   `json:"name"`
	IQN  string   `json:"iqn"`
	IPs  []net.IP `json:"ips"`

	Capacity struct {
		Size int    `json:"size,string"`
		Unit string `json:"unit"`
	} `json:"capacity"`
}
