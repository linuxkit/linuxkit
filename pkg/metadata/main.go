package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
)

const (
	// Hostname is the filename in configPath where the hostname is stored
	Hostname = "hostname"

	// SSH is the path where sshd configuration from the provider is stored
	SSH = "ssh"
)

var (
	defaultLogFormatter = &log.TextFormatter{}

	// ConfigPath is where the data is extracted to (a var so tests can override it)
	ConfigPath = "/run/config"

	// ec2Base is the AWS-compatible metadata service (a var so tests can override it)
	ec2Base = "http://169.254.169.254/"
)

// infoFormatter overrides the default format for Info() log events to
// provide an easier to read output
type infoFormatter struct {
}

func (f *infoFormatter) Format(entry *log.Entry) ([]byte, error) {
	if entry.Level == log.InfoLevel {
		return append([]byte(entry.Message), '\n'), nil
	}
	return defaultLogFormatter.Format(entry)
}

// Provider is a generic interface for metadata/userdata providers.
type Provider interface {
	// String should return a unique name for the Provider
	String() string

	// Probe returns true if the provider was detected.
	Probe(ctx context.Context) bool

	// Extract user data. This may write some data, specific to a
	// provider, to ConfigPath and should return the generic userdata.
	Extract(ctx context.Context) ([]byte, error)
}

// probe runs p.Probe with a fresh retry budget
func probe(p Provider) bool {
	ctx, cancel := withBudget(context.Background())
	defer cancel()
	return p.Probe(ctx)
}

// extract runs p.Extract with a fresh retry budget
func extract(p Provider) ([]byte, error) {
	ctx, cancel := withBudget(context.Background())
	defer cancel()
	return p.Extract(ctx)
}

var newProvider = map[string]func() Provider{
	"aws":          func() Provider { return NewAWS() },
	"gcp":          func() Provider { return NewGCP() },
	"hetzner":      func() Provider { return NewHetzner() },
	"openstack":    func() Provider { return NewOpenstack() },
	"scaleway":     func() Provider { return NewScaleway() },
	"vultr":        func() Provider { return NewVultr() },
	"digitalocean": func() Provider { return NewDigitalOcean() },
	"metaldata":    func() Provider { return NewMetalData() },
}

var newEC2 = func() Provider { return NewEC2() }
var newVMware = func() Provider {
	if p := NewVMware(); p != nil {
		return p
	}
	return nil
}
var listCDROMs = ListCDROMs

func mismatchWarning(plat platform, args []string) string {
	if plat.provider == "" {
		return ""
	}
	for _, arg := range args {
		if arg == plat.provider {
			return ""
		}
	}
	return fmt.Sprintf("metadata: DMI says %s, but the command line lists %s; using the command line", plat.provider, strings.Join(args, ", "))
}

func chooseProvider(args []string, plat platform) Provider {
	if len(args) > 0 {
		// Validate the whole list before constructing or probing any provider.
		for _, arg := range args {
			if newProvider[arg] == nil && arg != "vmware" && arg != "cdrom" && !strings.HasPrefix(arg, "file=") {
				log.Fatalf("Unrecognised metadata provider: %s", arg)
			}
		}
		if warning := mismatchWarning(plat, args); warning != "" {
			log.Warn(warning)
		}
		var network, local, files []Provider
		for _, arg := range args {
			switch {
			case newProvider[arg] != nil:
				network = append(network, newProvider[arg]())
			case arg == "vmware":
				if p := newVMware(); p != nil {
					local = append(local, p)
				}
			case arg == "cdrom":
				local = append(local, listCDROMs()...)
			case strings.HasPrefix(arg, "file="):
				files = append(files, fileProvider(arg[5:]))
			}
		}
		// Keep the existing network, local, then file precedence within a list.
		for group, providers := range [][]Provider{network, local, files} {
			for _, p := range providers {
				if group == 1 {
					log.Printf("Trying %s", p)
				} else if group == 2 {
					log.Printf("Trying file %s", p)
				}
				if probe(p) {
					return p
				}
			}
		}
		return nil
	}

	switch {
	case plat.provider != "":
		log.Infof("metadata: DMI identifies %s (%s)", plat.provider, plat.evidence)
		p := newProvider[plat.provider]()
		if probe(p) {
			return p
		}
		log.Errorf("metadata: DMI identifies %s but its metadata service did not answer; trying cdrom", plat.provider)
	case plat.name != "":
		log.Infof("metadata: detected %s (%s), no metadata provider", plat.name, plat.evidence)
	default:
		log.Infof("metadata: DMI unknown, probing aws/openstack, gcp, hetzner, scaleway, vultr, digitalocean, metaldata, vmware in parallel (%s)", budget)
		ranked := []Provider{
			newEC2(), newProvider["gcp"](), newProvider["hetzner"](),
			newProvider["scaleway"](), newProvider["vultr"](),
			newProvider["digitalocean"](), newProvider["metaldata"](),
		}
		ctx, cancel := withBudget(context.Background())
		p := probeParallel(ctx, ranked, newVMware())
		cancel()
		if p != nil {
			return p
		}
	}
	for _, p := range listCDROMs() {
		log.Printf("Trying %s", p)
		if probe(p) {
			return p
		}
	}
	return nil
}

func run(args []string, plat platform) (Provider, []byte, error) {
	p := chooseProvider(args, plat)
	if p == nil {
		return nil, nil, nil
	}
	log.Printf("%s: Probe succeeded", p)
	userdata, err := extract(p)
	return p, userdata, err
}

func main() {
	log.SetFormatter(new(infoFormatter))
	log.SetLevel(log.InfoLevel)
	flagVerbose := flag.Bool("v", false, "Verbose execution")

	flag.Parse()
	if *flagVerbose {
		// Switch back to the standard formatter
		log.SetFormatter(defaultLogFormatter)
		log.SetLevel(log.DebugLevel)
	}

	if err := os.MkdirAll(ConfigPath, 0755); err != nil {
		log.Fatalf("Could not create %s: %s", ConfigPath, err)
	}

	p, userdata, err := run(flag.Args(), identify())
	if p == nil {
		log.Printf("No metadata/userdata found. Bye")
		return
	}

	if err != nil {
		log.Printf("Error during metadata probe: %s", err)
	}

	err = os.WriteFile(path.Join(ConfigPath, "provider"), []byte(p.String()), 0644)
	if err != nil {
		log.Printf("Error writing metadata provider: %s", err)
	}

	if userdata != nil {
		if err := processUserData(ConfigPath, userdata); err != nil {
			log.Printf("Could not extract user data: %s", err)
		}
	}

	// Handle setting the hostname as a special case. We want to
	// do this early and don't really want another container for it.
	hostname, err := os.ReadFile(path.Join(ConfigPath, Hostname))
	if err == nil {
		name := kernelHostname(string(hostname))
		if name != strings.TrimSpace(string(hostname)) {
			log.Printf("Hostname %q does not fit the kernel, using %q", strings.TrimSpace(string(hostname)), name)
		}
		err := syscall.Sethostname([]byte(name))
		if err != nil {
			log.Printf("Failed to set hostname: %s", err)
		} else {
			log.Printf("Set hostname to: %s", name)
		}
	}
}

// kernelHostname makes name fit the kernel's 64-byte hostname limit: a name
// too long keeps its first label, cut to 63 bytes (a DNS label's limit) if needed
func kernelHostname(name string) string {
	name = strings.TrimSpace(name)
	if len(name) > 64 {
		name, _, _ = strings.Cut(name, ".")
		if len(name) > 63 {
			name = name[:63]
		}
	}
	return name
}

// If the userdata is a json file, create a directory/file hierarchy.
// Example:
//
//	{
//	   "foobar" : {
//	       "foo" : {
//	           "perm": "0644",
//	           "content": "hello"
//	       }
//	}
//
// Will create foobar/foo with mode 0644 and content "hello"
func processUserData(basePath string, data []byte) error {
	// Always write the raw data to a file
	err := os.WriteFile(path.Join(basePath, "userdata"), data, 0644)
	if err != nil {
		log.Printf("Could not write userdata: %s", err)
		return err
	}

	var root ConfigFile
	if err := json.Unmarshal(data, &root); err != nil {
		// Userdata is no JSON, presumably...
		log.Printf("Could not unmarshall userdata: %s", err)
		// This is not an error
		return nil
	}

	for dir, entry := range root {
		writeConfigFiles(path.Join(basePath, dir), entry)
	}
	return nil
}

func writeConfigFiles(target string, current Entry) {
	if isFile(current) {
		filemode, err := parseFileMode(current.Perm, 0644)
		if err != nil {
			log.Printf("Failed to parse permission %+v: %s", current, err)
			return
		}
		if err := os.WriteFile(target, []byte(*current.Content), filemode); err != nil {
			log.Printf("Failed to write %s: %s", target, err)
			return
		}
	} else if isDirectory(current) {
		filemode, err := parseFileMode(current.Perm, 0755)
		if err != nil {
			log.Printf("Failed to parse permission %+v: %s", current, err)
			return
		}
		if err := os.MkdirAll(target, filemode); err != nil {
			log.Printf("Failed to create %s: %s", target, err)
			return
		}
		for dir, entry := range current.Entries {
			writeConfigFiles(path.Join(target, dir), entry)
		}
	} else {
		log.Printf("%s is invalid", target)
	}
}

func isFile(json Entry) bool {
	return json.Content != nil && json.Entries == nil
}

func isDirectory(json Entry) bool {
	return json.Content == nil && json.Entries != nil
}

func parseFileMode(input string, defaultMode os.FileMode) (os.FileMode, error) {
	if input != "" {
		perm, err := strconv.ParseUint(input, 8, 32)
		if err != nil {
			return 0, err
		}
		return os.FileMode(perm), nil
	}
	return defaultMode, nil
}

// ConfigFile represents the configuration file
type ConfigFile map[string]Entry

// Entry represents either a directory or a file
type Entry struct {
	Perm    string           `json:"perm,omitempty"`
	Content *string          `json:"content,omitempty"`
	Entries map[string]Entry `json:"entries,omitempty"`
}
