package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

var (
	dmiRoot            = "/sys/class/dmi/id"
	hypervisorUUIDPath = "/sys/hypervisor/uuid"
	instanceIDRe       = regexp.MustCompile(`^i-([0-9a-f]{8}|[0-9a-f]{17})$`)
)

// platform is the cloud the DMI/SMBIOS tables identify
type platform struct {
	name     string // "aws", "azure", …; "" when unknown
	provider string // provider name for the command-line registry; "" when none
	evidence string // file=%q of the matching value, e.g. sys_vendor="Amazon EC2"
}

// readTrimmed returns the trimmed contents of path, or "" if it cannot be read
func readTrimmed(label, path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		log.Debugf("metadata: %s: %v", label, err)
		return ""
	}
	v := strings.TrimSpace(string(b))
	log.Debugf("metadata: %s=%q", label, v)
	return v
}

// identify matches the DMI tables against known clouds; first match wins
func identify() platform {
	dmi := map[string]string{}
	if _, err := os.Stat(dmiRoot); err != nil {
		log.Debugf("metadata: no DMI at %s: %v", dmiRoot, err)
	} else {
		for _, f := range []string{"sys_vendor", "product_name", "product_uuid", "product_serial", "board_asset_tag", "chassis_asset_tag"} {
			dmi[f] = readTrimmed(f, filepath.Join(dmiRoot, f))
		}
	}
	readTrimmed("hypervisor_uuid", hypervisorUUIDPath)
	found := func(name, provider, file, value string) platform {
		return platform{name, provider, fmt.Sprintf("%s=%q", file, value)}
	}
	openstack := map[string]bool{"OpenStack Nova": true, "OpenStack Compute": true}
	openstackTags := map[string]bool{"OpenStack Nova": true, "OpenStack Compute": true, "OpenTelekomCloud": true,
		"HUAWEICLOUD": true, "SAP CCloud VM": true, "Samsung Cloud Platform": true}
	vendors := map[string]string{"Scaleway": "scaleway", "Vultr": "vultr", "DigitalOcean": "digitalocean"}

	switch v, p, s, tag := dmi["sys_vendor"], dmi["product_name"], dmi["product_serial"], dmi["chassis_asset_tag"]; {
	case v == "Amazon EC2":
		return found("aws", "aws", "sys_vendor", v)
	case instanceIDRe.MatchString(dmi["board_asset_tag"]):
		return found("aws", "aws", "board_asset_tag", dmi["board_asset_tag"])
	case p == "Google Compute Engine":
		return found("gcp", "gcp", "product_name", p)
	case strings.HasPrefix(s, "GoogleCloud-"):
		return found("gcp", "gcp", "product_serial", s)
	case v == "Hetzner":
		return found("hetzner", "hetzner", "sys_vendor", v)
	case openstack[p]:
		return found("openstack", "openstack", "product_name", p)
	case openstackTags[tag]:
		return found("openstack", "openstack", "chassis_asset_tag", tag)
	case vendors[v] != "":
		return found(vendors[v], vendors[v], "sys_vendor", v)
	case tag == "7783-7084-3265-9085-8269-3286-77":
		return found("azure", "", "chassis_asset_tag", tag)
	case tag == "OracleCloud.com":
		return found("oracle", "", "chassis_asset_tag", tag)
	case p == "Alibaba Cloud ECS":
		return found("alibaba", "", "product_name", p)
	}
	// UUID prefixes also occur on non-AWS Xen guests; let the shared EC2 probe disambiguate them.
	return platform{}
}
