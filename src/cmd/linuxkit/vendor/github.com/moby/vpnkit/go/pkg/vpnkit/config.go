package vpnkit

import (
	"encoding/json"
	"io"

	"github.com/pkg/errors"
)

// DHCPConfiguration configures the built-in DHCP server.
type DHCPConfiguration struct {
	SearchDomains []string `json:"searchDomains"`
	DomainName    string   `json:"domainName"`
}

func (d DHCPConfiguration) Write(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(d); err != nil {
		return errors.Wrap(err, "while writing DHCPConfiguration")
	}
	return nil
}

// HTTPConfiguration configures the built-in HTTP proxy.
type HTTPConfiguration struct {
	HTTP                  string `json:"http,omitempty"`
	HTTPS                 string `json:"https,omitempty"`
	Exclude               string `json:"exclude,omitempty"`
	TransparentHTTPPorts  []int  `json:"transparent_http_ports"`
	TransparentHTTPSPorts []int  `json:"transparent_https_ports"`
}

func (h HTTPConfiguration) Write(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(h); err != nil {
		return errors.Wrap(err, "while writing HTTPConfiguration")
	}
	return nil
}

// Forward is a single forward from the gateway IP ExternalPort to (InternalIP, InternalPort)
type Forward struct {
	Protocol     Protocol `json:"protocol"`
	ExternalPort int      `json:"external_port"`
	InternalIP   string   `json:"internal_ip"`
	InternalPort int      `json:"internal_port"`
}

// GatewayForwards is a list of individual forwards.
type GatewayForwards []Forward

func (g GatewayForwards) Write(w io.Writer) error {
	enc := json.NewEncoder(w)
	if err := enc.Encode(g); err != nil {
		return errors.Wrap(err, "while writing GatewayForewards")
	}
	return nil
}
