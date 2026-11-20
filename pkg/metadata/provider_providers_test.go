package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// providerCase is one provider with the metadata paths its fake serves (relative
// to its base URL) and the files and userdata Extract must produce from them
type providerCase struct {
	name     string
	p        Provider
	base     *string
	flavor   bool // GCP: answer 403 without Metadata-Flavor: Google
	paths    map[string]string
	files    map[string]string
	userdata string
}

func providerCases() []providerCase {
	return []providerCase{
		{
			name: "gcp", p: NewGCP(), base: &gcpBase, flavor: true,
			paths: map[string]string{
				"instance/hostname":             "gcp-host",
				"project/attributes/ssh-keys":   "root:ssh-ed25519 GCP",
				"instance/attributes/user-data": "gcp-ud",
			},
			files:    map[string]string{Hostname: "gcp-host", "ssh/authorized_keys": "ssh-ed25519 GCP\n"},
			userdata: "gcp-ud",
		},
		{
			name: "openstack", p: NewOpenstack(), base: &ec2Base,
			paths: map[string]string{
				"latest/meta-data/hostname":                    "os-host",
				"latest/meta-data/public-ipv4":                 "1.2.3.4",
				"latest/meta-data/local-ipv4":                  "10.0.0.4",
				"latest/meta-data/placement/availability-zone": "nova",
				"latest/meta-data/instance-type":               "m1.small",
				"latest/meta-data/instance-id":                 "i-0001",
				"latest/meta-data/local-hostname":              "os-local",
				"latest/meta-data/public-keys/0/openssh-key":   "ssh-ed25519 OS",
				"latest/user-data":                             "os-ud",
			},
			files: map[string]string{
				Hostname: "os-host", "public_ipv4": "1.2.3.4", "local_ipv4": "10.0.0.4",
				"availability_zone": "nova", "instance_type": "m1.small", "instance_id": "i-0001",
				"local_hostname": "os-local", "ssh/authorized_keys": "ssh-ed25519 OS",
			},
			userdata: "os-ud",
		},
		{
			name: "scaleway", p: NewScaleway(), base: &scalewayBase,
			paths: map[string]string{
				"conf": "HOSTNAME=scw-host\nID=scw-id\nLOCATION_ZONE_ID=par1\nPUBLIC_IP_ADDRESS=1.2.3.5\n" +
					"PRIVATE_IP=10.0.0.5\nSSH_PUBLIC_KEYS=1\nSSH_PUBLIC_KEYS_0_KEY='ssh-ed25519 SCW'\n",
				"user_data/cloud-init": "scw-ud",
			},
			files: map[string]string{
				Hostname: "scw-host", "instance_id": "scw-id", "instance_location": "par1",
				"public_ip": "1.2.3.5", "private_ip": "10.0.0.5", "ssh/authorized_keys": "ssh-ed25519 SCW\n",
			},
			userdata: "scw-ud",
		},
		{
			name: "vultr", p: NewVultr(), base: &vultrBase,
			paths: map[string]string{
				"":                          "index",
				"hostname":                  "vultr-host",
				"interfaces/0/ipv4/address": "1.2.3.6",
				"interfaces/1/ipv4/address": "10.0.0.6",
				"interfaces/1/ipv4/netmask": "255.255.0.0",
				"region/regioncode":         "AMS",
				"instanceid":                "vultr-id",
				"public-keys":               "ssh-ed25519 VULTR",
			},
			files: map[string]string{
				Hostname: "vultr-host", "public_ipv4": "1.2.3.6", "private_ipv4": "10.0.0.6",
				"private_netmask": "255.255.0.0", "region_code": "AMS", "instance_id": "vultr-id",
				"ssh/authorized_keys": "ssh-ed25519 VULTR",
			},
		},
		{
			name: "digitalocean", p: NewDigitalOcean(), base: &digitalOceanBase,
			paths: map[string]string{
				"":                                  "index",
				"hostname":                          "do-host",
				"interfaces/public/0/ipv4/address":  "1.2.3.7",
				"interfaces/private/0/ipv4/address": "10.0.0.7",
				"region":                            "ams3",
				"id":                                "do-id",
				"public-keys":                       "ssh-ed25519 DO",
				"user-data":                         "do-ud",
			},
			files: map[string]string{
				Hostname: "do-host", "public_ipv4": "1.2.3.7", "private_ipv4": "10.0.0.7",
				"region": "ams3", "id": "do-id", "ssh/authorized_keys": "ssh-ed25519 DO",
			},
			userdata: "do-ud",
		},
		{
			name: "metaldata", p: NewMetalData(), base: &metaldataBase,
			paths: map[string]string{
				"meta/hostname":        "md-host",
				"meta/public-ipv4":     "1.2.3.8",
				"meta/private-ipv4":    "10.0.0.8",
				"meta/failure-domain":  "rack1",
				"meta/machine-id":      "md-id",
				"meta/machine-type":    "big",
				"meta/authorized-keys": "ssh-ed25519 MD",
				"user":                 "md-ud",
			},
			files: map[string]string{
				Hostname: "md-host", "public_ipv4": "1.2.3.8", "private_ipv4": "10.0.0.8",
				"failure_domain": "rack1", "machine_id": "md-id", "machine_type": "big",
				"ssh/authorized_keys": "ssh-ed25519 MD",
			},
			userdata: "md-ud",
		},
	}
}

// serveProvider points c's base URL and the Scaleway userdata address at a fake
// that drops the first drops requests, and ConfigPath at a temporary directory
func serveProvider(t *testing.T, c providerCase, drops int32) context.Context {
	srv, _ := flaky(t, drops, func(w http.ResponseWriter, r *http.Request) {
		if c.flavor && r.Header.Get("Metadata-Flavor") != "Google" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		body, found := c.paths[strings.TrimPrefix(r.URL.Path, "/")]
		if !found {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte(body))
	})
	fakeTime(t)
	base, cfg, ud := *c.base, ConfigPath, scalewayUserdataAddr
	*c.base, ConfigPath, scalewayUserdataAddr = srv.URL+"/", t.TempDir(), srv.Listener.Addr().String()
	ctx, cancel := withBudget(context.Background())
	t.Cleanup(func() {
		cancel()
		*c.base, ConfigPath, scalewayUserdataAddr = base, cfg, ud
	})
	return ctx
}

func TestProviderProbeExtract(t *testing.T) {
	for _, c := range providerCases() {
		t.Run(c.name, func(t *testing.T) {
			ctx := serveProvider(t, c, 0)
			if !c.p.Probe(ctx) {
				t.Fatal("Probe failed")
			}
			if c.name == "scaleway" && os.Geteuid() != 0 {
				c.userdata = "" // userdata needs a source port below 1024
			}
			userdata, err := c.p.Extract(ctx)
			if err != nil || string(userdata) != c.userdata {
				t.Fatalf("Extract: %q, %v; want %q", userdata, err, c.userdata)
			}
			for name, want := range c.files {
				if got, err := os.ReadFile(path.Join(ConfigPath, name)); err != nil || string(got) != want {
					t.Errorf("%s: %q, %v; want %q", name, got, err, want)
				}
			}
		})
	}
}

func TestProviderProbeRetriesThenAnswers(t *testing.T) {
	for _, c := range providerCases() {
		t.Run(c.name, func(t *testing.T) {
			ctx := serveProvider(t, c, 1)
			if !c.p.Probe(ctx) {
				t.Fatal("Probe failed")
			}
			checkSleeps(t, time.Second)
		})
	}
}

// scalewayFake points scalewayUserdataAddr at a fake that drops the first drops
// requests, then answers with code and "ud"; it records the client's source port
func scalewayFake(t *testing.T, drops int32, code int) (*atomic.Int32, *atomic.Int32) {
	if os.Geteuid() != 0 {
		t.Skip("binding a source port below 1024 needs root")
	}
	var port atomic.Int32
	srv, count := flaky(t, drops, func(w http.ResponseWriter, r *http.Request) {
		_, p, _ := net.SplitHostPort(r.RemoteAddr)
		n, _ := strconv.Atoi(p)
		port.Store(int32(n))
		w.WriteHeader(code)
		w.Write([]byte("ud"))
	})
	fakeTime(t)
	old := scalewayUserdataAddr
	scalewayUserdataAddr = srv.Listener.Addr().String()
	t.Cleanup(func() { scalewayUserdataAddr = old })
	return count, &port
}

func TestScalewayUserdataRetries(t *testing.T) {
	count, port := scalewayFake(t, 1, http.StatusOK)
	ctx, cancel := withBudget(context.Background())
	defer cancel()
	if ud, err := scalewayGetUserdata(ctx); err != nil || string(ud) != "ud" {
		t.Fatalf("userdata: %q, %v", ud, err)
	}
	if count.Load() != 2 {
		t.Fatalf("want 2 requests, got %d", count.Load())
	}
	if p := port.Load(); p < 1 || p >= 1024 {
		t.Fatalf("want source port below 1024, got %d", p)
	}
	checkSleeps(t, time.Second)
}

func TestScalewayUserdata404Final(t *testing.T) {
	count, _ := scalewayFake(t, 0, http.StatusNotFound)
	ctx, cancel := withBudget(context.Background())
	defer cancel()
	if ud, err := scalewayGetUserdata(ctx); err != nil || ud != nil {
		t.Fatalf("want nil, nil; got %q, %v", ud, err)
	}
	if count.Load() != 1 {
		t.Fatalf("want 1 request, got %d", count.Load())
	}
	checkSleeps(t)
}

func TestScalewayUserdataDeadline(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("binding a source port below 1024 needs root")
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	fakeTime(t)
	old := scalewayUserdataAddr
	scalewayUserdataAddr = l.Addr().String()
	t.Cleanup(func() { scalewayUserdataAddr = old })
	l.Close()
	ctx, cancel := withBudget(context.Background())
	defer cancel()
	if _, err := scalewayGetUserdata(ctx); err == nil {
		t.Fatal("userdata succeeded with the listener closed")
	}
	s := time.Second
	checkSleeps(t, s, 2*s, 4*s, 5*s, 5*s, 5*s, 5*s, 3*s)
}

// hetznerFake serves paths (relative to the server root) and points hetznerBase
// and ec2Base at it; it counts requests outside /hetzner/v1/ and answers them 404
func hetznerFake(t *testing.T, paths map[string]string) (context.Context, *atomic.Int32) {
	var strays atomic.Int32
	srv, _ := flaky(t, 0, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/hetzner/v1/") {
			t.Logf("request outside /hetzner/v1/: %s", r.URL.Path)
			strays.Add(1)
		}
		body, found := paths[r.URL.Path]
		if !found {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte(body))
	})
	fakeTime(t)
	hb, eb, cfg := hetznerBase, ec2Base, ConfigPath
	hetznerBase = strings.Replace(hetznerBase, "http://169.254.169.254", srv.URL, 1)
	ec2Base, ConfigPath = srv.URL+"/", t.TempDir()
	ctx, cancel := withBudget(context.Background())
	t.Cleanup(func() {
		cancel()
		hetznerBase, ec2Base, ConfigPath = hb, eb, cfg
	})
	return ctx, &strays
}

func TestHetznerExtract(t *testing.T) {
	ctx, strays := hetznerFake(t, map[string]string{
		"/hetzner/v1/metadata/hostname":    "hz-host",
		"/hetzner/v1/metadata/instance-id": "42",
		"/hetzner/v1/metadata/public-ipv4": "1.2.3.9",
		"/hetzner/v1/metadata/public-keys": `["ssh-ed25519 AAAA one","ssh-rsa BBBB two"]`,
		"/hetzner/v1/userdata":             "hz-ud",
	})
	p := NewHetzner()
	if !p.Probe(ctx) {
		t.Fatal("Probe failed")
	}
	userdata, err := p.Extract(ctx)
	if err != nil || string(userdata) != "hz-ud" {
		t.Fatalf("Extract: %q, %v; want %q", userdata, err, "hz-ud")
	}
	for name, want := range map[string]string{
		Hostname: "hz-host", "instance_id": "42", "public_ipv4": "1.2.3.9",
		"ssh/authorized_keys": "ssh-ed25519 AAAA one\nssh-rsa BBBB two\n",
	} {
		if got, err := os.ReadFile(path.Join(ConfigPath, name)); err != nil || string(got) != want {
			t.Errorf("%s: %q, %v; want %q", name, got, err, want)
		}
	}
	if _, err := os.Stat(path.Join(ConfigPath, "local_ipv4")); !os.IsNotExist(err) {
		t.Errorf("local_ipv4 written: %v", err)
	}
	if n := strays.Load(); n != 0 {
		t.Errorf("%d requests outside /hetzner/v1/", n)
	}
}

func TestHetznerIgnoresEC2Routes(t *testing.T) {
	ctx, _ := hetznerFake(t, map[string]string{"/latest/meta-data/hostname": "ec2-host"})
	if NewHetzner().Probe(ctx) {
		t.Fatal("Probe succeeded on an EC2-only metadata service")
	}
	checkSleeps(t) // 404 is final, no retry
}
