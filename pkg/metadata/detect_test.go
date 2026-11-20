package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIdentify(t *testing.T) {
	tests := []struct {
		name       string
		dmi        map[string]string // nil: no DMI directory
		hypervisor string            // "": no /sys/hypervisor/uuid
		want       platform
	}{
		{"aws sys_vendor", map[string]string{"sys_vendor": "Amazon EC2\n"}, "",
			platform{"aws", "aws", `sys_vendor="Amazon EC2"`}},
		{"aws metal", map[string]string{"sys_vendor": "Amazon EC2", "product_name": "m5.metal"}, "",
			platform{"aws", "aws", `sys_vendor="Amazon EC2"`}},
		{"aws xen uuid", map[string]string{"sys_vendor": "Xen", "product_uuid": "EC2E1A45-0000-0000-0000-000000000000"}, "",
			platform{"aws", "aws", `product_uuid="EC2E1A45-0000-0000-0000-000000000000"`}},
		{"aws xen uuid little-endian", map[string]string{"sys_vendor": "Xen", "product_uuid": "45E12AEC-0000-0000-0000-000000000000"}, "",
			platform{"aws", "aws", `product_uuid="45E12AEC-0000-0000-0000-000000000000"`}},
		{"aws hypervisor uuid", nil, "ec2a1b2c-0000-0000-0000-000000000000\n",
			platform{"aws", "aws", `hypervisor_uuid="ec2a1b2c-0000-0000-0000-000000000000"`}},
		{"hypervisor uuid byte-swapped only", nil, "45e12aec-0000-0000-0000-000000000000\n", platform{}},
		{"vmware ec2 uuid", map[string]string{"sys_vendor": "VMware, Inc.", "product_uuid": "EC2E1A45-0000-0000-0000-000000000000"}, "", platform{}},
		{"vmware ec2 uuid little-endian", map[string]string{"sys_vendor": "VMware, Inc.", "product_uuid": "45E12AEC-0000-0000-0000-000000000000"}, "", platform{}},
		{"qemu ec2 uuid", map[string]string{"sys_vendor": "QEMU", "product_uuid": "ec2e1a45-0000-0000-0000-000000000000"}, "", platform{}},
		{"qemu ec2 uuid little-endian", map[string]string{"sys_vendor": "QEMU", "product_uuid": "45e12aec-0000-0000-0000-000000000000"}, "", platform{}},
		{"aws instance id 8", map[string]string{"board_asset_tag": "i-0123abcd"}, "",
			platform{"aws", "aws", `board_asset_tag="i-0123abcd"`}},
		{"instance id 12", map[string]string{"board_asset_tag": "i-0123456789ab"}, "", platform{}},
		{"aws instance id", map[string]string{"board_asset_tag": "i-0123456789abcdef0"}, "",
			platform{"aws", "aws", `board_asset_tag="i-0123456789abcdef0"`}},
		{"short instance id", map[string]string{"board_asset_tag": "i-123"}, "", platform{}},
		{"gcp product_name", map[string]string{"product_name": "Google Compute Engine"}, "",
			platform{"gcp", "gcp", `product_name="Google Compute Engine"`}},
		{"gcp product_serial", map[string]string{"product_serial": "GoogleCloud-ABC"}, "",
			platform{"gcp", "gcp", `product_serial="GoogleCloud-ABC"`}},
		{"hetzner", map[string]string{"sys_vendor": "Hetzner"}, "",
			platform{"hetzner", "hetzner", `sys_vendor="Hetzner"`}},
		{"openstack nova", map[string]string{"product_name": "OpenStack Nova"}, "",
			platform{"openstack", "openstack", `product_name="OpenStack Nova"`}},
		{"openstack compute", map[string]string{"product_name": "OpenStack Compute"}, "",
			platform{"openstack", "openstack", `product_name="OpenStack Compute"`}},
		{"scaleway", map[string]string{"sys_vendor": "Scaleway"}, "",
			platform{"scaleway", "scaleway", `sys_vendor="Scaleway"`}},
		{"vultr", map[string]string{"sys_vendor": "Vultr"}, "",
			platform{"vultr", "vultr", `sys_vendor="Vultr"`}},
		{"digitalocean", map[string]string{"sys_vendor": "DigitalOcean"}, "",
			platform{"digitalocean", "digitalocean", `sys_vendor="DigitalOcean"`}},
		{"azure", map[string]string{"chassis_asset_tag": "7783-7084-3265-9085-8269-3286-77"}, "",
			platform{"azure", "", `chassis_asset_tag="7783-7084-3265-9085-8269-3286-77"`}},
		{"oracle", map[string]string{"chassis_asset_tag": "OracleCloud.com"}, "",
			platform{"oracle", "", `chassis_asset_tag="OracleCloud.com"`}},
		{"alibaba", map[string]string{"product_name": "Alibaba Cloud ECS"}, "",
			platform{"alibaba", "", `product_name="Alibaba Cloud ECS"`}},
		{"vmware", map[string]string{"sys_vendor": "VMware, Inc.", "product_name": "VMware Virtual Platform"}, "", platform{}},
		{"qemu", map[string]string{"sys_vendor": "QEMU", "product_name": "Standard PC (Q35 + ICH9, 2009)"}, "", platform{}},
		{"no dmi", nil, "", platform{}},
		{"empty files", map[string]string{"sys_vendor": "", "product_name": "", "product_uuid": "", "product_serial": "",
			"board_asset_tag": "", "chassis_asset_tag": ""}, "", platform{}},
		{"precedence", map[string]string{"sys_vendor": "Amazon EC2", "product_name": "Google Compute Engine"}, "",
			platform{"aws", "aws", `sys_vendor="Amazon EC2"`}},
	}
	for _, tag := range []string{"OpenStack Nova", "OpenStack Compute", "OpenTelekomCloud", "HUAWEICLOUD", "SAP CCloud VM", "Samsung Cloud Platform"} {
		tests = append(tests, struct {
			name       string
			dmi        map[string]string
			hypervisor string
			want       platform
		}{"openstack tag " + tag, map[string]string{"chassis_asset_tag": tag}, "",
			platform{"openstack", "openstack", `chassis_asset_tag="` + tag + `"`}})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			oldRoot, oldUUID := dmiRoot, hypervisorUUIDPath
			t.Cleanup(func() { dmiRoot, hypervisorUUIDPath = oldRoot, oldUUID })
			dmiRoot = filepath.Join(dir, "dmi")
			hypervisorUUIDPath = filepath.Join(dir, "uuid")
			if tt.dmi != nil {
				if err := os.Mkdir(dmiRoot, 0o755); err != nil {
					t.Fatal(err)
				}
				for f, v := range tt.dmi {
					if err := os.WriteFile(filepath.Join(dmiRoot, f), []byte(v), 0o644); err != nil {
						t.Fatal(err)
					}
				}
			}
			if tt.hypervisor != "" {
				if err := os.WriteFile(hypervisorUUIDPath, []byte(tt.hypervisor), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if got := identify(); got != tt.want {
				t.Errorf("identify() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestIsEC2UUID(t *testing.T) {
	for s, want := range map[string]bool{
		"ec2e1a45-0000-0000-0000-000000000000": true,
		"EC2E1A45-0000-0000-0000-000000000000": true,
		"45E12AEC-0000-0000-0000-000000000000": true,
		"12345678-0000-0000-0000-000000000000": false,
		"45e1":                                 false,
		"":                                     false,
	} {
		if got := isEC2UUID(s); got != want {
			t.Errorf("isEC2UUID(%q) = %v, want %v", s, got, want)
		}
	}
}
