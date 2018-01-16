package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
	"syscall"
)

const (
	// ConfigPath is where the data is extracted to
	ConfigPath = "/run/config"

	// Hostname is the filename in configPath where the hostname is stored
	Hostname = "hostname"

	// SSH is the path where sshd configuration from the provider is stored
	SSH = "ssh"

	// Standard AWS-compatible Metadata URLs
	userDataURL = "http://169.254.169.254/latest/user-data"
	metaDataURL = "http://169.254.169.254/latest/meta-data/"
)

// Provider is a generic interface for metadata/userdata providers.
type Provider interface {
	// String should return a unique name for the Provider
	String() string

	// Probe returns true if the provider was detected.
	Probe() bool

	// Extract user data. This may write some data, specific to a
	// provider, to ConfigPath and should return the generic userdata.
	Extract() ([]byte, error)
}

// netProviders is a list of Providers offering metadata/userdata over the network
var netProviders []Provider

// cdromProviders is a list of Providers offering metadata/userdata data via CDROM
var cdromProviders []Provider

func main() {
	providers := []string{"aws", "gcp", "openstack", "vultr", "packet", "cdrom"}
	if len(os.Args) > 1 {
		providers = os.Args[1:]
	}
	for _, p := range providers {
		switch p {
		case "aws":
			netProviders = append(netProviders, NewAWS())
		case "gcp":
			netProviders = append(netProviders, NewGCP())
		case "openstack":
			netProviders = append(netProviders, NewOpenstack())
		case "packet":
			netProviders = append(netProviders, NewPacket())
		case "vultr":
			netProviders = append(netProviders, NewVultr())
		case "cdrom":
			cdromProviders = ListCDROMs()
		default:
			log.Fatalf("Unrecognised metadata provider: %s", p)
		}
	}

	if err := os.MkdirAll(ConfigPath, 0755); err != nil {
		log.Fatalf("Could not create %s: %s", ConfigPath, err)
	}

	var p Provider
	var userdata []byte
	var err error
	found := false
	for _, p = range netProviders {
		if p.Probe() {
			log.Printf("%s: Probe succeeded", p)
			userdata, err = p.Extract()
			found = true
			break
		}
	}
	if !found {
		for _, p = range cdromProviders {
			log.Printf("Trying %s", p.String())
			if p.Probe() {
				log.Printf("%s: Probe succeeded", p)
				userdata, err = p.Extract()
				found = true
				break
			}
		}
	}

	if !found {
		log.Printf("No metadata/userdata found. Bye")
		return
	}

	if err != nil {
		log.Printf("Error during metadata probe: %s", err)
	}

	err = ioutil.WriteFile(path.Join(ConfigPath, "provider"), []byte(p.String()), 0644)
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
	hostname, err := ioutil.ReadFile(path.Join(ConfigPath, Hostname))
	if err == nil {
		err := syscall.Sethostname(hostname)
		if err != nil {
			log.Printf("Failed to set hostname: %s", err)
		} else {
			log.Printf("Set hostname to: %s", string(hostname))
		}
	}
}

// If the userdata is a json file, create a directory/file hierarchy.
// Example:
// {
//    "foobar" : {
//        "foo" : {
//            "perm": "0644",
//            "content": "hello"
//        }
// }
// Will create foobar/foo with mode 0644 and content "hello"
func processUserData(basePath string, data []byte) error {
	// Always write the raw data to a file
	err := ioutil.WriteFile(path.Join(basePath, "userdata"), data, 0644)
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
		if err := ioutil.WriteFile(target, []byte(*current.Content), filemode); err != nil {
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
