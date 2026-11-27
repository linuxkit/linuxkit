package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
)

func TestSampleConfig(t *testing.T) {
	basePath, err := os.MkdirTemp("", "metadata")
	if err != nil {
		t.Fatalf("can't make a temp rootdir %v", err)
	}
	defer os.RemoveAll(basePath)

	process(t, basePath, `{
	  "ssh": {
		"entries": {
		  "sshd_config": {
			"perm": "0600",
			"content": "PermitRootLogin yes\nPasswordAuthentication no"
		  }
		}
	  },
	  "foo": {
		"entries": {
		  "bar": {
			"content": "foobar"
		  },
		  "baz": {
			"perm": "0600",
			"content": "bar"
		  }
		}
	  }
	}`)

	sshd := path.Join(basePath, "ssh", "sshd_config")
	assertContent(t, sshd, "PermitRootLogin yes\nPasswordAuthentication no")
	assertPermission(t, sshd, 0600)

	bar := path.Join(basePath, "foo", "bar")
	assertContent(t, bar, "foobar")
	assertPermission(t, bar, 0644)

	assertContent(t, path.Join(basePath, "foo", "baz"), "bar")
}

func TestSerialization(t *testing.T) {
	bin, err := json.Marshal(ConfigFile{
		"ssh": Entry{
			Entries: map[string]Entry{
				"sshd_config": {
					Content: str("PermitRootLogin yes\nPasswordAuthentication no"),
					Perm:    "0600",
				},
			},
		},
		"foo": Entry{
			Entries: map[string]Entry{
				"bar": {
					Content: str("foobar"),
				},
				"baz": {
					Content: str("bar"),
					Perm:    "0600",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Cannot convert to json: %v", err)
	}

	expected := `{"foo":{"entries":{"bar":{"content":"foobar"},"baz":{"perm":"0600","content":"bar"}}},"ssh":{"entries":{"sshd_config":{"perm":"0600","content":"PermitRootLogin yes\nPasswordAuthentication no"}}}}`
	if expected != string(bin) {
		t.Fatalf("Expected %v but has %v", expected, string(bin))
	}
}

func TestWriteSingleFile(t *testing.T) {
	basePath, err := os.MkdirTemp(os.TempDir(), "metadata")
	if err != nil {
		t.Fatalf("can't make a temp rootdir %v", err)
	}
	defer os.RemoveAll(basePath)

	process(t, basePath, `{
	  "hostname": {
		"content": "foobar"
	  }
	}`)

	assertContent(t, path.Join(basePath, "hostname"), "foobar")
}

func TestWriteEmptyFile(t *testing.T) {
	basePath, err := os.MkdirTemp(os.TempDir(), "metadata")
	if err != nil {
		t.Fatalf("can't make a temp rootdir %v", err)
	}
	defer os.RemoveAll(basePath)

	process(t, basePath, `{
	  "empty": {
		"content": ""
	  }
	}`)

	assertContent(t, path.Join(basePath, "empty"), "")
}

func TestWriteEmptyDirectory(t *testing.T) {
	basePath, err := os.MkdirTemp(os.TempDir(), "metadata")
	if err != nil {
		t.Fatalf("can't make a temp rootdir %v", err)
	}
	defer os.RemoveAll(basePath)

	process(t, basePath, `{
	  "empty": {
		"entries": {}
	  }
	}`)

	if _, err := os.Stat(path.Join(basePath, "empty")); err != nil {
		t.Fatalf("empty folder doesn't exist: %v", err)
	}
}

func TestSetPermission(t *testing.T) {
	basePath, err := os.MkdirTemp(os.TempDir(), "metadata")
	if err != nil {
		t.Fatalf("can't make a temp rootdir %v", err)
	}
	defer os.RemoveAll(basePath)

	process(t, basePath, `{
	  "restricted": {
		"perm": "0600",
		"entries": {
		  "password": {
			"perm": "0600",
			"content": "secret"
		  }
		}
	  }
	}`)

	assertPermission(t, path.Join(basePath, "restricted"), 0600|os.ModeDir)
	assertPermission(t, path.Join(basePath, "restricted", "password"), 0600)
}

func TestDeepTree(t *testing.T) {
	basePath, err := os.MkdirTemp("", "metadata")
	if err != nil {
		t.Fatalf("can't make a temp rootdir %v", err)
	}
	defer os.RemoveAll(basePath)

	process(t, basePath, `{
	  "level1": {
		"entries": {
		  "level2": {
			"entries": {
			  "file2": {
				"content": "depth2"
			  },
			  "level3": {
				"entries": {
				  "file3": {
					"content": "depth3"
				  }
				}
			  }
			}
		  }
		}
	  }
	}`)

	assertContent(t, path.Join(basePath, "level1", "level2", "level3", "file3"), "depth3")
	assertContent(t, path.Join(basePath, "level1", "level2", "file2"), "depth2")
}

func str(input string) *string {
	return &input
}

func process(t *testing.T, basePath string, json string) {
	if err := processUserData(basePath, []byte(json)); err != nil {
		t.Fatalf("fail to process json %v", err)
	}
}

func assertPermission(t *testing.T, path string, expected os.FileMode) {
	fileinfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("%v doesn't exist: %v", path, err)
	}
	if fileinfo.Mode() != expected {
		t.Fatalf("%v: expected %v but has %v", path, expected, fileinfo.Mode())
	}
}

func assertContent(t *testing.T, path, expected string) {
	file, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("can't read %v: %v", path, err)
	}
	if !bytes.Equal(file, []byte(expected)) {
		t.Fatalf("%v: expected %v but has %v", path, expected, string(file))
	}
}

func TestKernelHostname(t *testing.T) {
	for in, want := range map[string]string{
		"host\n":                          "host",
		strings.Repeat("a", 63):           strings.Repeat("a", 63),
		strings.Repeat("a", 64):           strings.Repeat("a", 64),
		"inst." + strings.Repeat("b", 60): "inst",
		strings.Repeat("c", 70):           strings.Repeat("c", 63),
	} {
		if got := kernelHostname(in); got != want {
			t.Errorf("kernelHostname(%q) = %q; want %q", in, got, want)
		}
	}
}

type chooseFakes struct {
	providers    map[string]*fakeProvider
	constructors map[string]int
}

func fakeChoices(t *testing.T) *chooseFakes {
	t.Helper()
	oldProviders, oldEC2, oldVMware, oldCDROMs := newProvider, newEC2, newVMware, listCDROMs
	t.Cleanup(func() {
		newProvider, newEC2, newVMware, listCDROMs = oldProviders, oldEC2, oldVMware, oldCDROMs
	})
	f := &chooseFakes{make(map[string]*fakeProvider), make(map[string]int)}
	constructor := func(name string) func() Provider {
		p := &fakeProvider{name: name, done: make(chan struct{})}
		f.providers[name] = p
		return func() Provider {
			f.constructors[name]++
			return p
		}
	}
	newProvider = make(map[string]func() Provider)
	for _, name := range []string{"aws", "gcp", "hetzner", "openstack", "scaleway", "vultr", "digitalocean", "metaldata"} {
		newProvider[name] = constructor(name)
	}
	newEC2, newVMware = constructor("ec2"), constructor("vmware")
	cdrom := constructor("cdrom")
	listCDROMs = func() []Provider { return []Provider{cdrom()} }
	return f
}

func (f *chooseFakes) onlyConstructed(t *testing.T, names ...string) {
	t.Helper()
	want := make(map[string]int)
	for _, name := range names {
		want[name]++
	}
	for name := range f.providers {
		if got := f.constructors[name]; got != want[name] {
			t.Errorf("%s constructors = %d; want %d", name, got, want[name])
		}
	}
}

func (f *chooseFakes) awaitParallel(t *testing.T) {
	t.Helper()
	for _, name := range []string{"ec2", "gcp", "hetzner", "scaleway", "vultr", "digitalocean", "metaldata", "vmware"} {
		select {
		case <-f.providers[name].done:
		case <-time.After(5 * time.Second):
			t.Fatalf("%s probe did not finish", name)
		}
		if got := f.providers[name].probes.Load(); got != 1 {
			t.Errorf("%s probes = %d; want 1", name, got)
		}
	}
}

func TestChooseKnownProbesOnlyThatProvider(t *testing.T) {
	f := fakeChoices(t)
	logs := captureProbeLogs(t)
	f.providers["aws"].ok = true
	plat := platform{name: "aws", provider: "aws", evidence: `sys_vendor="Amazon EC2"`}
	if got := chooseProvider(nil, plat); got != f.providers["aws"] {
		t.Fatalf("provider = %v; want aws", got)
	}
	f.onlyConstructed(t, "aws")
	if got := f.providers["aws"].probes.Load(); got != 1 {
		t.Errorf("aws probes = %d; want 1", got)
	}
	if !strings.Contains(logs.String(), `metadata: DMI identifies aws (sys_vendor=\"Amazon EC2\")`) {
		t.Fatalf("missing DMI decision in %s", logs.String())
	}
}

func TestChooseKnownFailureGoesToCDROM(t *testing.T) {
	f := fakeChoices(t)
	logs := captureProbeLogs(t)
	f.providers["cdrom"].ok = true
	if got := chooseProvider(nil, platform{name: "aws", provider: "aws"}); got != f.providers["cdrom"] {
		t.Fatalf("provider = %v; want cdrom", got)
	}
	f.onlyConstructed(t, "aws", "cdrom")
	if f.providers["aws"].probes.Load() != 1 || f.providers["cdrom"].probes.Load() != 1 {
		t.Fatal("known provider and cdrom must each be probed once")
	}
	checkProbeConflict(t, logs, "metadata: DMI identifies aws but its metadata service did not answer; trying cdrom")
}

func TestChooseKnownNoProvider(t *testing.T) {
	f := fakeChoices(t)
	logs := captureProbeLogs(t)
	f.providers["cdrom"].ok = true
	if got := chooseProvider(nil, platform{name: "azure", evidence: `chassis_asset_tag="7783"`}); got != f.providers["cdrom"] {
		t.Fatalf("provider = %v; want cdrom", got)
	}
	f.onlyConstructed(t, "cdrom")
	if !strings.Contains(logs.String(), `metadata: detected azure (chassis_asset_tag=\"7783\"), no metadata provider`) {
		t.Fatalf("missing unsupported-cloud decision in %s", logs.String())
	}
}

func TestChooseUnknownProbesAll(t *testing.T) {
	f := fakeChoices(t)
	logs := captureProbeLogs(t)
	f.providers["vmware"].ok = true
	got := chooseProvider(nil, platform{})
	f.awaitParallel(t)
	if got != f.providers["vmware"] {
		t.Fatalf("provider = %v; want vmware", got)
	}
	f.onlyConstructed(t, "ec2", "gcp", "hetzner", "scaleway", "vultr", "digitalocean", "metaldata", "vmware")
	if !strings.Contains(logs.String(), "metadata: DMI unknown, probing aws/openstack, gcp, hetzner, scaleway, vultr, digitalocean, metaldata, vmware in parallel (30s)") {
		t.Fatalf("missing parallel decision in %s", logs.String())
	}
}

func TestChooseXenUUIDOnlyReachesOpenStack(t *testing.T) {
	for _, uuid := range []string{"ec2e1a45-1234-4567-89ab-0123456789ab", "45e12aec-1234-4567-89ab-0123456789ab"} {
		t.Run(uuid, func(t *testing.T) {
			f := fakeChoices(t)
			oldRoot, oldUUID := dmiRoot, hypervisorUUIDPath
			dmiRoot, hypervisorUUIDPath = t.TempDir(), path.Join(t.TempDir(), "uuid")
			t.Cleanup(func() { dmiRoot, hypervisorUUIDPath = oldRoot, oldUUID })
			for name, value := range map[string]string{"sys_vendor": "Xen", "product_uuid": uuid} {
				if err := os.WriteFile(path.Join(dmiRoot, name), []byte(value), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(hypervisorUUIDPath, []byte("ec2a1b2c-0000-0000-0000-000000000000"), 0600); err != nil {
				t.Fatal(err)
			}
			var disambiguations atomic.Int32
			probeIMDS(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/latest/api/token":
					w.WriteHeader(http.StatusMethodNotAllowed)
				case "/latest/meta-data/hostname":
					w.Write([]byte("openstack-host"))
				case "/openstack/latest/meta_data.json":
					disambiguations.Add(1)
					w.Write([]byte(`{}`))
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			})
			newEC2 = func() Provider { return NewEC2() }
			got := chooseProvider(nil, identify())
			for _, name := range []string{"gcp", "hetzner", "scaleway", "vultr", "digitalocean", "metaldata", "vmware"} {
				select {
				case <-f.providers[name].done:
				case <-time.After(5 * time.Second):
					t.Fatalf("%s probe did not finish", name)
				}
			}
			if got == nil || got.String() != "openstack" || disambiguations.Load() != 1 {
				t.Fatalf("provider = %v, disambiguations = %d; want openstack, 1", got, disambiguations.Load())
			}
		})
	}
}

type chooseContextProvider struct {
	*fakeProvider
	probeCtx, extractCtx context.Context
}

func (p *chooseContextProvider) Probe(ctx context.Context) bool {
	p.probeCtx = ctx
	return p.fakeProvider.Probe(ctx)
}

func (p *chooseContextProvider) Extract(ctx context.Context) ([]byte, error) {
	p.extractCtx = ctx
	return p.fakeProvider.Extract(ctx)
}

func TestChooseUnknownExtractFreshContext(t *testing.T) {
	f := fakeChoices(t)
	p := &chooseContextProvider{fakeProvider: f.providers["ec2"]}
	p.ok = true
	newEC2 = func() Provider {
		f.constructors["ec2"]++
		return p
	}
	got, _, err := run(nil, platform{})
	f.awaitParallel(t)
	if got != p || err != nil {
		t.Fatalf("run = %v, %v; want ec2, nil", got, err)
	}
	if p.extractCtx == nil || p.ctxErr != nil {
		t.Fatalf("Extract did not receive a live context: %v", p.ctxErr)
	}
	if p.probeCtx == p.extractCtx || p.probeCtx.Err() != context.Canceled {
		t.Fatal("Extract reused the parallel probe context or probes were not cancelled")
	}
	if deadline, ok := p.extractCtx.Deadline(); !ok || time.Until(deadline) < budget-time.Second {
		t.Fatal("Extract did not receive a fresh full retry budget")
	}
}

func TestChooseExplicitListWins(t *testing.T) {
	f := fakeChoices(t)
	logs := captureProbeLogs(t)
	f.providers["cdrom"].ok = true
	if got := chooseProvider([]string{"gcp", "cdrom"}, platform{name: "aws", provider: "aws"}); got != f.providers["cdrom"] {
		t.Fatalf("provider = %v; want cdrom", got)
	}
	f.onlyConstructed(t, "gcp", "cdrom")
	if f.providers["gcp"].probes.Load() != 1 || f.providers["aws"].probes.Load() != 0 {
		t.Fatal("explicit list did not override DMI")
	}
	if !strings.Contains(logs.String(), "level=warning") || !strings.Contains(logs.String(), "metadata: DMI says aws, but the command line lists gcp, cdrom; using the command line") {
		t.Fatalf("missing mismatch warning in %s", logs.String())
	}
}

func TestChooseExplicitSequentialOrder(t *testing.T) {
	f := fakeChoices(t)
	f.providers["vultr"].ok, f.providers["gcp"].ok = true, true
	if got := chooseProvider([]string{"vultr", "gcp"}, platform{}); got != f.providers["vultr"] {
		t.Fatalf("provider = %v; want vultr", got)
	}
	if f.providers["gcp"].probes.Load() != 0 {
		t.Fatal("later provider was probed after a success")
	}
}

func TestChooseExplicitLocalOrder(t *testing.T) {
	for _, vmwareOK := range []bool{false, true} {
		t.Run(strconv.FormatBool(vmwareOK), func(t *testing.T) {
			f := fakeChoices(t)
			f.providers["vmware"].ok = vmwareOK
			f.providers["cdrom"].ok = true
			filename := path.Join(t.TempDir(), "userdata")
			if err := os.WriteFile(filename, []byte("local data"), 0600); err != nil {
				t.Fatal(err)
			}
			got := chooseProvider([]string{"file=" + filename, "vmware", "cdrom", "gcp"}, platform{})
			want := f.providers["cdrom"]
			if vmwareOK {
				want = f.providers["vmware"]
			}
			if got != want || f.providers["gcp"].probes.Load() != 1 {
				t.Fatalf("provider = %v; want %v after network probe", got, want)
			}
		})
	}
}

func TestChooseExplicitFileFallback(t *testing.T) {
	f := fakeChoices(t)
	filename := path.Join(t.TempDir(), "userdata")
	if err := os.WriteFile(filename, []byte("local data"), 0600); err != nil {
		t.Fatal(err)
	}
	got, data, err := run([]string{"file=" + filename, "cdrom", "aws"}, platform{name: "gcp", provider: "gcp"})
	if got != fileProvider(filename) || string(data) != "local data" || err != nil {
		t.Fatalf("run = %v, %q, %v; want file, local data, nil", got, data, err)
	}
	f.onlyConstructed(t, "aws", "cdrom")
}

func TestChooseUnknownFailureGoesToCDROM(t *testing.T) {
	f := fakeChoices(t)
	f.providers["cdrom"].ok = true
	got := chooseProvider(nil, platform{})
	f.awaitParallel(t)
	if got != f.providers["cdrom"] {
		t.Fatalf("provider = %v; want cdrom", got)
	}
}

func TestChooseExplicitNilVMware(t *testing.T) {
	f := fakeChoices(t)
	newVMware = func() Provider { return nil }
	f.providers["cdrom"].ok = true
	if got := chooseProvider([]string{"vmware", "cdrom"}, platform{}); got != f.providers["cdrom"] {
		t.Fatalf("provider = %v; want cdrom after unavailable vmware", got)
	}
}

func TestChooseValidatesWholeList(t *testing.T) {
	for _, invalid := range []string{"invalid", "equinixmetal"} {
		t.Run(invalid, func(t *testing.T) {
			f := fakeChoices(t)
			logs := captureProbeLogs(t)
			f.providers["aws"].ok = true
			logger := log.StandardLogger()
			oldExit := logger.ExitFunc
			t.Cleanup(func() { logger.ExitFunc = oldExit })
			logger.ExitFunc = func(code int) { panic(code) }
			defer func() {
				if got := recover(); got != 1 {
					t.Errorf("exit = %v; want 1", got)
				}
				f.onlyConstructed(t)
				if !strings.Contains(logs.String(), "Unrecognised metadata provider: "+invalid) {
					t.Errorf("missing invalid-provider error in %s", logs.String())
				}
			}()
			chooseProvider([]string{"aws", invalid}, platform{})
		})
	}
}

func TestChooseRunNoProvider(t *testing.T) {
	fakeChoices(t)
	p, data, err := run([]string{"aws"}, platform{})
	if p != nil || data != nil || err != nil {
		t.Fatalf("run = %v, %q, %v; want nil, nil, nil", p, data, err)
	}
}

func TestChooseRunExtractError(t *testing.T) {
	fakeChoices(t)
	filename := path.Join(t.TempDir(), "directory")
	if err := os.Mkdir(filename, 0700); err != nil {
		t.Fatal(err)
	}
	p, _, err := run([]string{"file=" + filename}, platform{})
	var pathErr *os.PathError
	if p != fileProvider(filename) || !errors.As(err, &pathErr) {
		t.Fatalf("run = %v, %v; want selected file and read error", p, err)
	}
}

func TestMismatchWarning(t *testing.T) {
	for _, tt := range []struct {
		name string
		plat platform
		args []string
		want string
	}{
		{"unknown", platform{}, []string{"aws"}, ""},
		{"unsupported", platform{name: "azure"}, []string{"aws"}, ""},
		{"listed", platform{name: "aws", provider: "aws"}, []string{"gcp", "aws"}, ""},
		{"mismatch", platform{name: "gcp", provider: "gcp"}, []string{"aws"}, "metadata: DMI says gcp, but the command line lists aws; using the command line"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := mismatchWarning(tt.plat, tt.args); got != tt.want {
				t.Errorf("mismatchWarning = %q; want %q", got, tt.want)
			}
		})
	}
}

func TestChooseExplicitFreshProbeBudgets(t *testing.T) {
	f := fakeChoices(t)
	first := &chooseContextProvider{fakeProvider: f.providers["gcp"]}
	second := &chooseContextProvider{fakeProvider: f.providers["vultr"]}
	second.ok = true
	newProvider["gcp"] = func() Provider { return first }
	newProvider["vultr"] = func() Provider { return second }
	if got := chooseProvider([]string{"gcp", "vultr"}, platform{}); got != second {
		t.Fatalf("provider = %v; want vultr", got)
	}
	if first.probeCtx == second.probeCtx {
		t.Fatal("sequential providers shared a probe budget")
	}
	for _, p := range []*chooseContextProvider{first, second} {
		if p.probeCtx == nil || p.probeCtx.Err() != context.Canceled {
			t.Fatalf("%s probe context was not released", p)
		}
		if deadline, ok := p.probeCtx.Deadline(); !ok || time.Until(deadline) < budget-time.Second {
			t.Fatalf("%s probe did not receive a full retry budget", p)
		}
	}
}

func TestChooseUnknownNilVMware(t *testing.T) {
	f := fakeChoices(t)
	newVMware = func() Provider { return nil }
	f.providers["ec2"].ok = true
	got := chooseProvider(nil, platform{})
	for _, name := range []string{"ec2", "gcp", "hetzner", "scaleway", "vultr", "digitalocean", "metaldata"} {
		select {
		case <-f.providers[name].done:
		case <-time.After(5 * time.Second):
			t.Fatalf("%s probe did not finish", name)
		}
	}
	if got != f.providers["ec2"] {
		t.Fatalf("provider = %v; want ec2 without vmware", got)
	}
}
